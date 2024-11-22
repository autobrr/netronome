package speedtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	st "github.com/showwin/speedtest-go/speedtest"

	"speedtrackerr/internal/database"
	"speedtrackerr/internal/types"
)

type Result struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Server        string    `json:"server"`
	DownloadSpeed float64   `json:"downloadSpeed"`
	UploadSpeed   float64   `json:"uploadSpeed"`
	Latency       string    `json:"latency"`
	PacketLoss    float64   `json:"packetLoss"`
	Jitter        float64   `json:"jitter"`
	Error         string    `json:"error,omitempty"`
	Download      float64   `json:"-"`
	Upload        float64   `json:"-"`
}

type Service interface {
	RunTest(opts *types.TestOptions) (*Result, error)
	GetServers() ([]ServerResponse, error)
}

type ServerResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Host     string  `json:"host"`
	Distance float64 `json:"distance"`
	Country  string  `json:"country"`
	Sponsor  string  `json:"sponsor"`
	URL      string  `json:"url"`
	Lat      float64 `json:"lat,string"`
	Lon      float64 `json:"lon,string"`
}

type Server struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Host            string  `json:"host"`
	Distance        float64 `json:"distance"`
	Country         string  `json:"country"`
	BroadcastUpdate func(types.SpeedUpdate)
}

type ProgressUpdate struct {
	ServerName   string  `json:"serverName"`
	TestType     string  `json:"testType"`
	CurrentSpeed float64 `json:"currentSpeed"`
}

type SpeedUpdate struct {
	Type        string  `json:"type"`
	ServerName  string  `json:"serverName"`
	Speed       float64 `json:"speed"`
	Progress    float64 `json:"progress"`
	IsComplete  bool    `json:"isComplete"`
	Latency     string  `json:"latency,omitempty"`
	IsScheduled bool    `json:"isScheduled"`
}

type service struct {
	client *st.Speedtest
	server *Server
	db     database.Service
}

func New(server *Server, db database.Service) Service {
	return &service{
		client: st.New(),
		server: server,
		db:     db,
	}
}

func (s *service) RunTest(opts *types.TestOptions) (*Result, error) {
	log.Debug().
		Bool("isScheduled", opts.IsScheduled).
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Msg("Starting speed test")

	serverList, err := s.client.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	var selectedServer *st.Server
	if len(opts.ServerIDs) > 0 {
		for _, server := range serverList {
			for _, requestedID := range opts.ServerIDs {
				if server.ID == requestedID {
					selectedServer = server
					break
				}
			}
			if selectedServer != nil {
				break
			}
		}
	}

	if selectedServer == nil {
		sort.Slice(serverList, func(i, j int) bool {
			return serverList[i].Distance < serverList[j].Distance
		})
		selectedServer = serverList[0]
	}

	log.Info().
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Str("server_name", selectedServer.Name).
		Str("server_host", selectedServer.Host).
		Str("server_country", selectedServer.Country).
		Str("provider", selectedServer.Sponsor).
		Bool("enable_download", opts.EnableDownload).
		Bool("enable_upload", opts.EnableUpload).
		Msg("Starting speed test")

	result := &Result{
		Timestamp: time.Now(),
		Server:    selectedServer.Name,
	}

	if err := selectedServer.PingTest(func(latency time.Duration) {
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
				Type:        "ping",
				ServerName:  selectedServer.Name,
				Latency:     latency.String(),
				Progress:    100,
				IsComplete:  false,
				IsScheduled: opts.IsScheduled,
			})
		}
	}); err != nil {
		result.Error = fmt.Sprintf("ping test failed: %v", err)
		return result, err
	}
	result.Latency = selectedServer.Latency.String()

	if s.server.BroadcastUpdate != nil {
		s.server.BroadcastUpdate(types.SpeedUpdate{
			Type:        "ping",
			ServerName:  selectedServer.Name,
			Latency:     selectedServer.Latency.String(),
			Progress:    100,
			IsComplete:  false,
			IsScheduled: opts.IsScheduled,
		})
	}

	if opts.EnableDownload {
		startTime := time.Now()
		expectedDuration := 10 * time.Second // Typical test duration

		selectedServer.Context.SetCallbackDownload(func(speed st.ByteRate) {
			if s.server.BroadcastUpdate != nil {
				elapsed := time.Since(startTime)
				progress := math.Min(100, (elapsed.Seconds()/expectedDuration.Seconds())*100)

				s.server.BroadcastUpdate(types.SpeedUpdate{
					Type:        "download",
					ServerName:  selectedServer.Name,
					Speed:       speed.Mbps(),
					Progress:    progress,
					IsComplete:  false,
					IsScheduled: opts.IsScheduled,
				})
			}
		})

		if err := selectedServer.DownloadTest(); err != nil {
			result.Error = fmt.Sprintf("download test failed: %v", err)
			return result, err
		}
		result.Download = selectedServer.DLSpeed.Mbps()
		result.DownloadSpeed = result.Download
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
				Type:        "download",
				ServerName:  selectedServer.Name,
				Speed:       result.DownloadSpeed,
				Progress:    1.0,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
			})
		}

		log.Info().
			Str("server", selectedServer.Name).
			Str("server_host", selectedServer.Host).
			Str("server_country", selectedServer.Country).
			Str("provider", selectedServer.Sponsor).
			Str("server_url", selectedServer.URL).
			Float64("speed_mbps", result.DownloadSpeed).
			Msg("Download test complete")
	}

	if opts.EnableUpload {
		startTime := time.Now()
		expectedDuration := 10 * time.Second // Typical test duration

		selectedServer.Context.SetCallbackUpload(func(speed st.ByteRate) {
			if s.server.BroadcastUpdate != nil {
				elapsed := time.Since(startTime)
				progress := math.Min(100, (elapsed.Seconds()/expectedDuration.Seconds())*100)

				s.server.BroadcastUpdate(types.SpeedUpdate{
					Type:        "upload",
					ServerName:  selectedServer.Name,
					Speed:       speed.Mbps(),
					Progress:    progress,
					IsComplete:  false,
					IsScheduled: opts.IsScheduled,
				})
			}
		})

		if err := selectedServer.UploadTestContext(context.Background()); err != nil {
			result.Error = fmt.Sprintf("upload test failed: %v", err)
			return result, err
		}
		result.Upload = selectedServer.ULSpeed.Mbps()
		result.UploadSpeed = result.Upload

		log.Info().
			Str("server", selectedServer.Name).
			Str("server_host", selectedServer.Host).
			Str("server_country", selectedServer.Country).
			Str("provider", selectedServer.Sponsor).
			Str("server_url", selectedServer.URL).
			Float64("speed_mbps", result.UploadSpeed).
			Msg("Upload test complete")

		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
				Type:        "upload",
				ServerName:  selectedServer.Name,
				Speed:       result.UploadSpeed,
				Progress:    1.0,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
			})
		}
	}

	log.Info().
		Str("server", selectedServer.Name).
		Str("server_host", selectedServer.Host).
		Str("server_country", selectedServer.Country).
		Str("provider", selectedServer.Sponsor).
		Str("server_url", selectedServer.URL).
		Str("latency", result.Latency).
		Float64("packet_loss", result.PacketLoss).
		Float64("download_mbps", result.DownloadSpeed).
		Float64("upload_mbps", result.UploadSpeed).
		Msg("Speed test complete")

	selectedServer.Context.Reset()

	// Update the database save operation
	jitterFloat := selectedServer.Jitter.Seconds() * 1000
	dbResult, err := s.db.SaveSpeedTest(context.Background(), database.SpeedTestResult{
		ServerName:    selectedServer.Name,
		ServerID:      selectedServer.ID,
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		PacketLoss:    result.PacketLoss,
		Jitter:        &jitterFloat,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to save result to database")
	}

	if dbResult != nil {
		result.ID = dbResult.ID
	}

	return result, nil
}

func (s *service) GetServers() ([]ServerResponse, error) {
	// Create new speedtest client
	client := st.New()

	// Get user info first to initialize the client
	_, err := client.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Fetch servers using the initialized client
	serverList, err := client.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	availableServers := serverList.Available()
	if availableServers == nil {
		return nil, fmt.Errorf("no available servers found")
	}

	// Convert to response format
	response := make([]ServerResponse, len(*availableServers))
	for i, server := range *availableServers {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)

		response[i] = ServerResponse{
			ID:       server.ID,
			Name:     server.Name,
			Host:     server.Host,
			Distance: server.Distance,
			Country:  server.Country,
			Sponsor:  server.Sponsor,
			URL:      server.URL,
			Lat:      lat,
			Lon:      lon,
		}
	}

	// Sort by distance
	sort.Slice(response, func(i, j int) bool {
		return response[i].Distance < response[j].Distance
	})

	log.Info().Int("server_count", len(response)).Msg("Retrieved speedtest servers")
	return response, nil
}
