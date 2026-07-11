// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/types"
)

type UrlDownloadRunner struct {
	progressCallback func(types.SpeedUpdate)
}

func NewUrlDownloadRunner() *UrlDownloadRunner {
	return &UrlDownloadRunner{}
}

func (r *UrlDownloadRunner) SetProgressCallback(callback func(types.SpeedUpdate)) {
	r.progressCallback = callback
}

func (r *UrlDownloadRunner) GetTestType() string {
	return "url_download"
}

func (r *UrlDownloadRunner) GetServers() ([]ServerResponse, error) {
	return builtinDownloadServers, nil
}

func (r *UrlDownloadRunner) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	downloadURL := opts.DownloadURL

	// If downloadURL is empty, try to get from serverIDs
	if downloadURL == "" && len(opts.ServerIDs) > 0 {
		for _, server := range builtinDownloadServers {
			if server.ID == opts.ServerIDs[0] {
				downloadURL = server.URL
				opts.ServerName = server.Name
				break
			}
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("download URL is required")
	}

	// Validate URL
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme must be http or https")
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("URL host is empty")
	}
	if len(downloadURL) > 2048 {
		return nil, fmt.Errorf("URL too long (max 2048 characters)")
	}

	// If ServerName is still empty (e.g., from custom URL), use the hostname
	if opts.ServerName == "" {
		opts.ServerName = parsedURL.Host
	}

	threads := opts.DownloadThreads
	if threads == 0 {
		threads = 4 // default
	}
	if threads != 2 && threads != 4 && threads != 8 {
		return nil, fmt.Errorf("download threads must be 2, 4, or 8")
	}

	timeout := opts.DownloadTimeout
	if timeout == 0 {
		timeout = 30 // default 30 seconds
	}

	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	log.Info().
		Str("url", downloadURL).
		Int("threads", threads).
		Int("timeout", timeout).
		Msg("Starting URL download test")

	startTime := time.Now()
	var ttfb time.Duration
	var totalBytes int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstByteOnce sync.Once
	errChan := make(chan error, threads)

	// Progress reporting goroutine
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-progressDone:
				return
			case <-ticker.C:
				mu.Lock()
				elapsed := time.Since(startTime).Seconds()
				currentSpeed := 0.0
				if elapsed > 0 {
					currentSpeed = float64(totalBytes) * 8 / elapsed / 1000000 // Mbps
				}
				mu.Unlock()

				if r.progressCallback != nil {
					r.progressCallback(types.SpeedUpdate{
						Type:        "download",
						ServerName:  opts.ServerName,
						Speed:       currentSpeed,
						Progress:    50.0,
						IsComplete:  false,
						IsScheduled: opts.IsScheduled,
						TestType:    "url_download",
					})
				}
			}
		}
	}()

	// Launch download workers
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			req, err := http.NewRequestWithContext(testCtx, "GET", downloadURL, nil)
			if err != nil {
				errChan <- fmt.Errorf("worker %d: failed to create request: %w", workerID, err)
				return
			}

			// Set User-Agent to avoid 403 errors from CDNs like Cloudflare
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "*/*")
			req.Header.Set("Accept-Encoding", "gzip, deflate")
			req.Header.Set("Connection", "keep-alive")

			// Set Referer header to the root URL (scheme + host)
			if parsedURL, err := url.Parse(downloadURL); err == nil {
				referer := fmt.Sprintf("%s://%s/", parsedURL.Scheme, parsedURL.Host)
				req.Header.Set("Referer", referer)
			}

			client := &http.Client{
				Timeout: time.Duration(timeout) * time.Second,
			}

			workerStart := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				errChan <- fmt.Errorf("worker %d: request failed: %w", workerID, err)
				return
			}
			defer resp.Body.Close()

			// Record TTFB for the first worker
			firstByteOnce.Do(func() {
				ttfb = time.Since(workerStart)
				log.Debug().
					Int("worker", workerID).
					Dur("ttfb", ttfb).
					Msg("First byte received")
			})

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
				errChan <- fmt.Errorf("worker %d: HTTP %d", workerID, resp.StatusCode)
				return
			}

			// Read response body and discard to memory
			buf := make([]byte, 32*1024)
			for {
				select {
				case <-testCtx.Done():
					return
				default:
					n, err := resp.Body.Read(buf)
					if n > 0 {
						mu.Lock()
						totalBytes += int64(n)
						mu.Unlock()
					}
					if err != nil {
						if err == io.EOF {
							return
						}
						log.Debug().
							Err(err).
							Int("worker", workerID).
							Msg("Read error")
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(progressDone)
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// Calculate results even if there were some errors
	elapsed := time.Since(startTime).Seconds()
	if elapsed == 0 {
		elapsed = 0.001
	}

	downloadSpeed := float64(totalBytes) * 8 / elapsed / 1000000 // Mbps
	latency := fmt.Sprintf("%.2fms", float64(ttfb.Microseconds())/1000.0)

	// Check for connection errors (no data received at all)
	if totalBytes == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("url_download test failed: %v", errors[0])
		}
		return nil, fmt.Errorf("url_download test failed: unable to connect or no data received")
	}

	// Timeout handling: if we got data, return partial results (normal timeout)
	// Only fail if we got connection errors without any data
	wasTimeout := ctx.Err() == context.DeadlineExceeded || testCtx.Err() == context.DeadlineExceeded

	result := &Result{
		Timestamp:     time.Now(),
		Server:        opts.ServerName,
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   0,
		Latency:       latency,
		Jitter:        0,
	}

	if wasTimeout {
		log.Warn().
			Float64("download_mbps", downloadSpeed).
			Str("latency", latency).
			Int64("total_bytes", totalBytes).
			Float64("elapsed_sec", elapsed).
			Msg("URL download test completed with timeout (partial result saved)")
	} else {
		log.Info().
			Float64("download_mbps", downloadSpeed).
			Str("latency", latency).
			Int64("total_bytes", totalBytes).
			Float64("elapsed_sec", elapsed).
			Msg("URL download test completed")
	}

	// Send final progress update
	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:        "download",
			ServerName:  opts.ServerName,
			Speed:       downloadSpeed,
			Progress:    100.0,
			IsComplete:  true,
			Latency:     latency,
			IsScheduled: opts.IsScheduled,
			TestType:    "url_download",
		})
	}

	return result, nil
}
