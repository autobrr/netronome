package speedtest

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
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
	Error         string    `json:"error,omitempty"`
	Download      float64   `json:"-"`
	Upload        float64   `json:"-"`
}

type Service interface {
	RunTest(opts *types.TestOptions) (*Result, error)
	GetHistory() []Result
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
	Type       string  `json:"type"`
	ServerName string  `json:"serverName"`
	Speed      float64 `json:"speed"`
	Progress   float64 `json:"progress"`
	IsComplete bool    `json:"isComplete"`
	Latency    string  `json:"latency,omitempty"`
}

type service struct {
	client  *st.Speedtest
	history []Result
	mu      sync.RWMutex
	server  *Server
	db      database.Service
}

func New(server *Server, db database.Service) Service {
	return &service{
		client:  st.New(),
		history: make([]Result, 0),
		server:  server,
		db:      db,
	}
}

func (s *service) RunTest(opts *types.TestOptions) (*Result, error) {
	serverList, err := s.client.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	var selectedServer *st.Server
	if len(opts.ServerIDs) > 0 {
		for _, server := range serverList {
			if server.ID == opts.ServerIDs[0] {
				selectedServer = server
				break
			}
		}
	}

	if selectedServer == nil {
		selectedServer = serverList[0]
	}

	result := &Result{
		Timestamp: time.Now(),
		Server:    selectedServer.Name,
	}

	if err := selectedServer.PingTest(func(latency time.Duration) {
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
				Type:       "ping",
				ServerName: selectedServer.Name,
				Latency:    latency.String(),
				Progress:   100,
				IsComplete: false,
			})
		}
	}); err != nil {
		result.Error = fmt.Sprintf("ping test failed: %v", err)
		return result, err
	}
	result.Latency = selectedServer.Latency.String()

	if s.server.BroadcastUpdate != nil {
		s.server.BroadcastUpdate(types.SpeedUpdate{
			Type:       "ping",
			ServerName: selectedServer.Name,
			Latency:    selectedServer.Latency.String(),
			Progress:   100,
			IsComplete: false,
		})
	}

	if opts.EnableDownload {
		selectedServer.Context.SetCallbackDownload(func(speed st.ByteRate) {
			if s.server.BroadcastUpdate != nil {
				s.server.BroadcastUpdate(types.SpeedUpdate{
					Type:       "download",
					ServerName: selectedServer.Name,
					Speed:      speed.Mbps(),
					Progress:   0.5,
					IsComplete: false,
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
				Type:       "download",
				ServerName: selectedServer.Name,
				Speed:      result.DownloadSpeed,
				Progress:   1.0,
				IsComplete: true,
			})
		}
	}

	if opts.EnableUpload {
		selectedServer.Context.SetCallbackUpload(func(speed st.ByteRate) {
			if s.server.BroadcastUpdate != nil {
				s.server.BroadcastUpdate(types.SpeedUpdate{
					Type:       "upload",
					ServerName: selectedServer.Name,
					Speed:      speed.Mbps(),
					Progress:   0.5,
					IsComplete: false,
				})
			}
		})

		if err := selectedServer.UploadTestContext(context.Background()); err != nil {
			result.Error = fmt.Sprintf("upload test failed: %v", err)
			return result, err
		}
		result.Upload = selectedServer.ULSpeed.Mbps()
		result.UploadSpeed = result.Upload

		log.Info().Msg("Upload test complete, sending final update")
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
				Type:       "upload",
				ServerName: selectedServer.Name,
				Speed:      result.UploadSpeed,
				Progress:   1.0,
				IsComplete: true,
			})
		}
	}

	log.Info().Msg("All tests complete, sending final status")
	if s.server.BroadcastUpdate != nil {
		s.server.BroadcastUpdate(types.SpeedUpdate{
			Type:       "complete",
			ServerName: selectedServer.Name,
			Speed:      0,
			Progress:   1.0,
			IsComplete: true,
		})
	}

	selectedServer.Context.Reset()

	// Update the database save operation
	dbResult, err := s.db.SaveSpeedTest(context.Background(), database.SpeedTestResult{
		ServerName:    selectedServer.Name,
		ServerID:      selectedServer.ID,
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		PacketLoss:    result.PacketLoss,
	})
	if err != nil {
		log.Error().Err(err).Str("type", fmt.Sprintf("%T", err)).Str("message", err.Error()).Msg("Failed to save result to database")
	}

	// Add database ID to result if save was successful
	if dbResult != nil {
		result.ID = dbResult.ID
	}

	// Add to in-memory history
	s.mu.Lock()
	s.history = append(s.history, *result)
	s.mu.Unlock()

	return result, nil
}

func (s *service) GetServers() ([]ServerResponse, error) {
	serverList, err := s.client.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	availableServers := serverList.Available()
	if availableServers == nil {
		return nil, fmt.Errorf("no available servers found")
	}

	sort.Slice(*availableServers, func(i, j int) bool {
		return (*availableServers)[i].Distance < (*availableServers)[j].Distance
	})

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

	return response, nil
}

func (s *service) GetHistory() []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.history
}
