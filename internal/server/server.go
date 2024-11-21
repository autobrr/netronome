package server

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"

	"speedtrackerr/internal/database"
	"speedtrackerr/internal/scheduler"
	"speedtrackerr/internal/speedtest"
	"speedtrackerr/internal/types"
)

type Server struct {
	Router     *gin.Engine
	speedtest  speedtest.Service
	db         database.Service
	scheduler  scheduler.Service
	mu         sync.RWMutex
	lastUpdate *types.SpeedUpdate
}

func NewServer(speedtest speedtest.Service, db database.Service, scheduler scheduler.Service) *Server {
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	s := &Server{
		Router:     router,
		speedtest:  speedtest,
		db:         db,
		scheduler:  scheduler,
		lastUpdate: &types.SpeedUpdate{},
	}

	s.RegisterRoutes()
	return s
}

func (s *Server) BroadcastUpdate(update types.SpeedUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUpdate = &update
	log.Printf("Broadcasting update: %+v", s.lastUpdate)
}

func (s *Server) RegisterRoutes() {
	api := s.Router.Group("/api")
	{
		api.GET("/servers", s.handleGetServers)
		api.POST("/speedtest", s.handleSpeedTest)
		api.GET("/speedtest/status", s.handleSpeedTestStatus)
		api.GET("/speedtest/history", s.handleGetSpeedTestHistory)
		api.GET("/schedules", s.handleGetSchedules)
		api.POST("/schedules", s.handleCreateSchedule)
		api.PUT("/schedules/:id", s.handleUpdateSchedule)
		api.DELETE("/schedules/:id", s.handleDeleteSchedule)
	}
}

// Handler methods
func (s *Server) handleSpeedTest(c *gin.Context) {
	var opts types.TestOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Reset lastUpdate before starting new test
	s.mu.Lock()
	s.lastUpdate = &types.SpeedUpdate{}
	s.mu.Unlock()

	result, err := s.speedtest.RunTest(&opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ensure final update is set
	s.mu.Lock()
	s.lastUpdate.IsComplete = true
	s.mu.Unlock()

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleSpeedTestHistory(c *gin.Context) {
	history := s.speedtest.GetHistory()
	c.JSON(http.StatusOK, history)
}

func (s *Server) handleGetServers(c *gin.Context) {
	servers, err := s.speedtest.GetServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

func (s *Server) handleSpeedTestStatus(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Sending status update: %+v", s.lastUpdate)
	c.JSON(http.StatusOK, s.lastUpdate)
}

func (s *Server) handleGetSpeedTestHistory(c *gin.Context) {
	ctx := c.Request.Context()
	history, err := s.db.GetSpeedTests(ctx, 100) // Get last 100 results
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (s *Server) handleGetSchedules(c *gin.Context) {
	schedules, err := s.db.GetSchedules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedules)
}

func (s *Server) handleCreateSchedule(c *gin.Context) {
	var schedule types.Schedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	log.Printf("Received schedule: %+v", schedule)

	createdSchedule, err := s.db.CreateSchedule(c.Request.Context(), schedule)
	if err != nil {
		log.Printf("Failed to create schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Created schedule: %+v", createdSchedule)
	c.JSON(http.StatusCreated, createdSchedule)
}

func (s *Server) handleUpdateSchedule(c *gin.Context) {
	var schedule types.Schedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err := s.db.UpdateSchedule(c.Request.Context(), schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated successfully"})
}

func (s *Server) handleDeleteSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	err = s.db.DeleteSchedule(c.Request.Context(), int64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}
