// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/utils"
)

const (
	// EnvPrefix is the prefix for all environment variables
	EnvPrefix = "NETRONOME__"

	// AppName is used for config directory naming
	AppName = "netronome"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	SQLite   DatabaseType = "sqlite"
	Postgres DatabaseType = "postgres"
)

// Config represents the application configuration
type Config struct {
	Database      DatabaseConfig     `toml:"database"`
	Server        ServerConfig       `toml:"server"`
	Logging       LoggingConfig      `toml:"logging"`
	Auth          AuthConfig         `toml:"auth"`
	OIDC          OIDCConfig         `toml:"oidc"`
	SpeedTest     SpeedTestConfig    `toml:"speedtest"`
	GeoIP         GeoIPConfig        `toml:"geoip"`
	Pagination    PaginationConfig   `toml:"pagination"`
	Session       SessionConfig      `toml:"session"`
	Notifications NotificationConfig `toml:"notifications"`
	PacketLoss    PacketLossConfig   `toml:"packetloss"`
	Agent         AgentConfig        `toml:"agent"`
	Monitor       MonitorConfig      `toml:"monitor"`
	Tailscale     TailscaleConfig    `toml:"tailscale"`
}

type DatabaseConfig struct {
	Type     DatabaseType `toml:"type" env:"DB_TYPE"`
	Host     string       `toml:"host" env:"DB_HOST"`
	Port     int          `toml:"port" env:"DB_PORT"`
	User     string       `toml:"user" env:"DB_USER"`
	Password string       `toml:"password" env:"DB_PASSWORD"`
	DBName   string       `toml:"dbname" env:"DB_NAME"`
	SSLMode  string       `toml:"sslmode" env:"DB_SSLMODE"`
	Path     string       `toml:"path" env:"DB_PATH"`
}

type ServerConfig struct {
	Host    string `toml:"host" env:"HOST"`
	Port    int    `toml:"port" env:"PORT"`
	BaseURL string `toml:"base_url" env:"BASE_URL"`
	GinMode string `toml:"gin_mode" env:"GIN_MODE"`
}

type LoggingConfig struct {
	Level string `toml:"level" env:"LOG_LEVEL"`
}

type AuthConfig struct {
	Whitelist []string `toml:"whitelist" env:"AUTH_WHITELIST"`
}

type OIDCConfig struct {
	Issuer       string `toml:"issuer" env:"OIDC_ISSUER"`
	ClientID     string `toml:"client_id" env:"OIDC_CLIENT_ID"`
	ClientSecret string `toml:"client_secret" env:"OIDC_CLIENT_SECRET"`
	RedirectURL  string `toml:"redirect_url" env:"OIDC_REDIRECT_URL"`
}

type SpeedTestConfig struct {
	IPerf      IperfConfig      `toml:"iperf"`
	Librespeed LibrespeedConfig `toml:"librespeed"`
	Timeout    int              `toml:"timeout" env:"SPEEDTEST_TIMEOUT"`
}

type IperfConfig struct {
	TestDuration  int        `toml:"test_duration" env:"IPERF_TEST_DURATION"`
	ParallelConns int        `toml:"parallel_conns" env:"IPERF_PARALLEL_CONNS"`
	Timeout       int        `toml:"timeout" env:"IPERF_TIMEOUT"`
	Ping          PingConfig `toml:"ping"`
}

type LibrespeedConfig struct {
	ServersPath string `toml:"-"`
	Timeout     int    `toml:"timeout" env:"LIBRESPEED_TIMEOUT"`
}

type PingConfig struct {
	Count    int `toml:"count" env:"IPERF_PING_COUNT"`
	Interval int `toml:"interval" env:"IPERF_PING_INTERVAL"`
	Timeout  int `toml:"timeout" env:"IPERF_PING_TIMEOUT"`
}

type PaginationConfig struct {
	DefaultPage      int    `toml:"default_page" env:"DEFAULT_PAGE"`
	DefaultPageSize  int    `toml:"default_page_size" env:"DEFAULT_PAGE_SIZE"`
	MaxPageSize      int    `toml:"max_page_size" env:"MAX_PAGE_SIZE"`
	DefaultTimeRange string `toml:"default_time_range" env:"DEFAULT_TIME_RANGE"`
	DefaultLimit     int    `toml:"default_limit" env:"DEFAULT_LIMIT"`
}

type SessionConfig struct {
	Secret string `toml:"session_secret" env:"SESSION_SECRET"`
}

type NotificationConfig struct {
	Enabled           bool    `toml:"enabled" env:"NOTIFICATIONS_ENABLED"`
	WebhookURL        string  `toml:"webhook_url" env:"NOTIFICATIONS_WEBHOOK_URL"`
	PingThreshold     float64 `toml:"ping_threshold" env:"NOTIFICATIONS_PING_THRESHOLD"`
	UploadThreshold   float64 `toml:"upload_threshold" env:"NOTIFICATIONS_UPLOAD_THRESHOLD"`
	DownloadThreshold float64 `toml:"download_threshold" env:"NOTIFICATIONS_DOWNLOAD_THRESHOLD"`
	DiscordMentionID  string  `toml:"discord_mention_id" env:"NOTIFICATIONS_DISCORD_MENTION_ID"`
}

type GeoIPConfig struct {
	CountryDatabasePath string `toml:"country_database_path" env:"GEOIP_COUNTRY_DATABASE_PATH"`
	ASNDatabasePath     string `toml:"asn_database_path" env:"GEOIP_ASN_DATABASE_PATH"`
}

type PacketLossConfig struct {
	Enabled                  bool `toml:"enabled" env:"PACKETLOSS_ENABLED"`
	DefaultInterval          int  `toml:"default_interval" env:"PACKETLOSS_DEFAULT_INTERVAL"`
	DefaultPacketCount       int  `toml:"default_packet_count" env:"PACKETLOSS_DEFAULT_PACKET_COUNT"`
	MaxConcurrentMonitors    int  `toml:"max_concurrent_monitors" env:"PACKETLOSS_MAX_CONCURRENT_MONITORS"`
	PrivilegedMode           bool `toml:"privileged_mode" env:"PACKETLOSS_PRIVILEGED_MODE"`
	RestoreMonitorsOnStartup bool `toml:"restore_monitors_on_startup" env:"PACKETLOSS_RESTORE_MONITORS_ON_STARTUP"`
}

type AgentConfig struct {
	Host         string   `toml:"host" env:"AGENT_HOST"`
	Port         int      `toml:"port" env:"AGENT_PORT"`
	Interface    string   `toml:"interface" env:"AGENT_INTERFACE"`
	APIKey       string   `toml:"api_key" env:"AGENT_API_KEY"`
	DiskIncludes []string `toml:"disk_includes" env:"AGENT_DISK_INCLUDES" envSeparator:","`
	DiskExcludes []string `toml:"disk_excludes" env:"AGENT_DISK_EXCLUDES" envSeparator:","`
}

type MonitorConfig struct {
	Enabled           bool   `toml:"enabled" env:"MONITOR_ENABLED"`
	ReconnectInterval string `toml:"reconnect_interval" env:"MONITOR_RECONNECT_INTERVAL"`
}

type TailscaleConfig struct {
	// Core settings
	Enabled    bool   `toml:"enabled" env:"TAILSCALE_ENABLED"`
	Method     string `toml:"method" env:"TAILSCALE_METHOD"`       // "auto", "host", or "tsnet"
	AuthKey    string `toml:"auth_key" env:"TAILSCALE_AUTH_KEY"`   // Required for tsnet mode
	
	// TSNet-specific settings
	Hostname   string `toml:"hostname" env:"TAILSCALE_HOSTNAME"`     // Custom hostname (optional)
	Ephemeral  bool   `toml:"ephemeral" env:"TAILSCALE_EPHEMERAL"`   // Remove on shutdown
	StateDir   string `toml:"state_dir" env:"TAILSCALE_STATE_DIR"`   // State directory
	ControlURL string `toml:"control_url" env:"TAILSCALE_CONTROL_URL"` // For Headscale
	
	// Agent settings
	AgentPort  int    `toml:"agent_port" env:"TAILSCALE_AGENT_PORT"` // Port for agent to listen on
	
	// Server discovery settings
	AutoDiscover      bool   `toml:"auto_discover" env:"TAILSCALE_AUTO_DISCOVER"`
	DiscoveryInterval string `toml:"discovery_interval" env:"TAILSCALE_DISCOVERY_INTERVAL"`
	DiscoveryPort     int    `toml:"discovery_port" env:"TAILSCALE_DISCOVERY_PORT"`
	DiscoveryPrefix   string `toml:"discovery_prefix" env:"TAILSCALE_DISCOVERY_PREFIX"`
	
	// Deprecated fields (for backward compatibility)
	PreferHost     bool                   `toml:"prefer_host" env:"TAILSCALE_PREFER_HOST"`
	Agent          TailscaleAgentConfig   `toml:"agent"`
	Monitor        TailscaleMonitorConfig `toml:"monitor"`
}

// Deprecated - kept for backward compatibility
type TailscaleAgentConfig struct {
	Enabled       bool `toml:"enabled" env:"TAILSCALE_AGENT_ENABLED"`
	AcceptRoutes  bool `toml:"accept_routes" env:"TAILSCALE_AGENT_ACCEPT_ROUTES"`
	Port          int  `toml:"port" env:"TAILSCALE_AGENT_PORT"`
}

// Deprecated - kept for backward compatibility
type TailscaleMonitorConfig struct {
	AutoDiscover      bool   `toml:"auto_discover" env:"TAILSCALE_MONITOR_AUTO_DISCOVER"`
	DiscoveryInterval string `toml:"discovery_interval" env:"TAILSCALE_MONITOR_DISCOVERY_INTERVAL"`
	DiscoveryPrefix   string `toml:"discovery_prefix" env:"TAILSCALE_MONITOR_DISCOVERY_PREFIX"`
	DiscoveryPort     int    `toml:"discovery_port" env:"TAILSCALE_MONITOR_DISCOVERY_PORT"`
}

func isRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	if _, err := os.Stat("/dev/.lxc-boot-id"); err == nil {
		return true
	}

	if os.Getpid() == 1 {
		return true
	}

	if user := os.Getenv("USERNAME"); user == "ContainerAdministrator" || user == "ContainerUser" {
		return true
	}

	if pd, _ := os.Open("/proc/1/cgroup"); pd != nil {
		defer pd.Close()
		b := make([]byte, 4096)
		pd.Read(b)
		if strings.Contains(string(b), "/docker") || strings.Contains(string(b), "/lxc") {
			return true
		}
	}

	return false
}

// New creates a new Config instance with default values
func New() *Config {
	return &Config{
		Database: DatabaseConfig{
			Type:    SQLite,
			Host:    "localhost",
			Port:    5432,
			User:    "postgres",
			DBName:  "netronome",
			SSLMode: "disable",
			Path:    "netronome.db",
		},
		Server: ServerConfig{
			Host:    "127.0.0.1",
			Port:    7575,
			BaseURL: "/",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		SpeedTest: SpeedTestConfig{
			IPerf: IperfConfig{
				TestDuration:  10,
				ParallelConns: 4,
				Timeout:       60,
				Ping: PingConfig{
					Count:    5,
					Interval: 1000,
					Timeout:  10,
				},
			},
			Librespeed: LibrespeedConfig{
				ServersPath: "librespeed-servers.json",
				Timeout:     60,
			},
			Timeout: 30,
		},
		Pagination: PaginationConfig{
			DefaultPage:      1,
			DefaultPageSize:  20,
			MaxPageSize:      100,
			DefaultTimeRange: "1w",
			DefaultLimit:     20,
		},
		Session: SessionConfig{
			Secret: "",
		},
		GeoIP: GeoIPConfig{
			CountryDatabasePath: "",
			ASNDatabasePath:     "",
		},
		Notifications: NotificationConfig{
			Enabled:           false,
			WebhookURL:        "",
			PingThreshold:     30,
			UploadThreshold:   200,
			DownloadThreshold: 200,
			DiscordMentionID:  "",
		},
		PacketLoss: PacketLossConfig{
			Enabled:                  true,
			DefaultInterval:          3600,
			DefaultPacketCount:       10,
			MaxConcurrentMonitors:    10,
			PrivilegedMode:           true,
			RestoreMonitorsOnStartup: false,
		},
		Agent: AgentConfig{
			Host:         "0.0.0.0",
			Port:         8200,
			Interface:    "",
			DiskIncludes: []string{},
			DiskExcludes: []string{},
		},
		Monitor: MonitorConfig{
			Enabled:           true,
			ReconnectInterval: "30s",
		},
		Tailscale: TailscaleConfig{
			Enabled:           false,
			Method:            "auto",
			AuthKey:           "",
			Hostname:          "",
			Ephemeral:         false,
			StateDir:          "~/.config/netronome/tsnet",
			ControlURL:        "",
			AgentPort:         8200,
			AutoDiscover:      true,
			DiscoveryInterval: "5m",
			DiscoveryPort:     8200,
			DiscoveryPrefix:   "",
			// Deprecated fields - kept for compatibility during migration
			Agent: TailscaleAgentConfig{
				Enabled:      false,
				AcceptRoutes: true,
				Port:         8200,
			},
			Monitor: TailscaleMonitorConfig{
				AutoDiscover:      true,
				DiscoveryInterval: "5m",
				DiscoveryPrefix:   "",
				DiscoveryPort:     8200,
			},
		},
	}
}

// Load loads the configuration from a TOML file and environment variables
func Load(configPath string) (*Config, error) {
	cfg := New()

	// If specific config path provided, only try that one
	if configPath != "" {
		if _, err := toml.DecodeFile(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to decode config file %s: %w", configPath, err)
		}
		if cfg.SpeedTest.Librespeed.Timeout == 0 {
			cfg.SpeedTest.Librespeed.Timeout = 60
		}
		log.Info().
			Str("path", configPath).
			Msg("Loaded configuration file")

		// If db path is relative, make it relative to config file
		if !filepath.IsAbs(cfg.Database.Path) {
			cfg.Database.Path = filepath.Join(filepath.Dir(configPath), cfg.Database.Path)
		}
		cfg.SpeedTest.Librespeed.ServersPath = filepath.Join(filepath.Dir(configPath), "librespeed-servers.json")
	} else {
		// Try each default path in order
		found := false
		paths := DefaultConfigPaths()
		for _, path := range paths {
			log.Debug().
				Str("checking_path", path).
				Msg("Checking for config file")

			if _, err := os.Stat(path); err == nil {
				log.Debug().
					Str("found_at", path).
					Msg("Found config file")

				if _, err := toml.DecodeFile(path, cfg); err == nil {
					if cfg.SpeedTest.Librespeed.Timeout == 0 {
						cfg.SpeedTest.Librespeed.Timeout = 60
					}
					log.Info().
						Str("path", path).
						Msg("Loaded configuration file")
					found = true

					// If db path is relative, make it relative to config file
					if !filepath.IsAbs(cfg.Database.Path) {
						cfg.Database.Path = filepath.Join(filepath.Dir(path), cfg.Database.Path)
					}
					cfg.SpeedTest.Librespeed.ServersPath = filepath.Join(filepath.Dir(path), "librespeed-servers.json")
					break
				}
			}
		}
		if !found {
			log.Info().
				Msg("No configuration file found, running with default values. Use 'netronome generate-config' to create one")
		}
	}

	// Override with environment variables
	if err := cfg.loadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %w", err)
	}

	return cfg, nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() error {
	c.loadDatabaseFromEnv()
	c.loadServerFromEnv()
	c.loadLoggingFromEnv()
	c.loadAuthFromEnv()
	c.loadOIDCFromEnv()
	c.loadSpeedTestFromEnv()
	c.loadPaginationFromEnv()
	c.loadSessionFromEnv()
	c.loadGeoIPFromEnv()
	c.loadNotificationsFromEnv()
	c.loadPacketLossFromEnv()
	c.loadAgentFromEnv()
	c.loadMonitorFromEnv()
	c.loadTailscaleFromEnv()
	return nil
}

func (c *Config) loadDatabaseFromEnv() {
	if v := getEnv("DB_TYPE"); v != "" {
		c.Database.Type = DatabaseType(v)
	}
	if v := getEnv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := getEnv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := getEnv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := getEnv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := getEnv("DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := getEnv("DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}
	if v := getEnv("DB_PATH"); v != "" {
		c.Database.Path = v
	}
}

func (c *Config) loadServerFromEnv() {
	if v := getEnv("HOST"); v != "" {
		c.Server.Host = v
	}
	if v := getEnv("PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := getEnv("BASE_URL"); v != "" {
		c.Server.BaseURL = strings.Trim(v, `"'`)
	}
	if v := getEnv("GIN_MODE"); v != "" {
		c.Server.GinMode = v
	}
}

func (c *Config) loadLoggingFromEnv() {
	if v := getEnv("LOG_LEVEL"); v != "" {
		c.Logging.Level = strings.ToLower(v)
	}
}

func (c *Config) loadAuthFromEnv() {
	if v := getEnv("AUTH_WHITELIST"); v != "" {
		c.Auth.Whitelist = strings.Split(v, ",")
	}
}

func (c *Config) loadOIDCFromEnv() {
	if v := getEnv("OIDC_ISSUER"); v != "" {
		c.OIDC.Issuer = v
	}
	if v := getEnv("OIDC_CLIENT_ID"); v != "" {
		c.OIDC.ClientID = v
	}
	if v := getEnv("OIDC_CLIENT_SECRET"); v != "" {
		c.OIDC.ClientSecret = v
	}
	if v := getEnv("OIDC_REDIRECT_URL"); v != "" {
		c.OIDC.RedirectURL = v
	}
}

func (c *Config) loadSpeedTestFromEnv() {
	if v := getEnv("SPEEDTEST_TIMEOUT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.Timeout = val
		}
	}
	if v := getEnv("IPERF_TEST_DURATION"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.TestDuration = val
		}
	}
	if v := getEnv("IPERF_PARALLEL_CONNS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.ParallelConns = val
		}
	}
	if v := getEnv("IPERF_TIMEOUT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.Timeout = val
		}
	}
	if v := getEnv("IPERF_PING_COUNT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.Ping.Count = val
		}
	}
	if v := getEnv("IPERF_PING_INTERVAL"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.Ping.Interval = val
		}
	}
	if v := getEnv("IPERF_PING_TIMEOUT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.Ping.Timeout = val
		}
	}
	if v := getEnv("LIBRESPEED_TIMEOUT"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.Librespeed.Timeout = val
		}
	}
}

func (c *Config) loadPaginationFromEnv() {
	if v := getEnv("DEFAULT_PAGE"); v != "" {
		if page, err := strconv.Atoi(v); err == nil {
			c.Pagination.DefaultPage = page
		}
	}
	if v := getEnv("DEFAULT_PAGE_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			c.Pagination.DefaultPageSize = size
		}
	}
	if v := getEnv("MAX_PAGE_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			c.Pagination.MaxPageSize = size
		}
	}
	if v := getEnv("DEFAULT_TIME_RANGE"); v != "" {
		c.Pagination.DefaultTimeRange = v
	}
	if v := getEnv("DEFAULT_LIMIT"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil {
			c.Pagination.DefaultLimit = limit
		}
	}
}

func (c *Config) loadSessionFromEnv() {
	if v := getEnv("SESSION_SECRET"); v != "" {
		c.Session.Secret = v
	}
}

func (c *Config) loadGeoIPFromEnv() {
	if v := getEnv("GEOIP_COUNTRY_DATABASE_PATH"); v != "" {
		c.GeoIP.CountryDatabasePath = v
	}
	if v := getEnv("GEOIP_ASN_DATABASE_PATH"); v != "" {
		c.GeoIP.ASNDatabasePath = v
	}
}

func (c *Config) loadNotificationsFromEnv() {
	if v := getEnv("NOTIFICATIONS_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.Notifications.Enabled = enabled
		}
	}
	if v := getEnv("NOTIFICATIONS_WEBHOOK_URL"); v != "" {
		c.Notifications.WebhookURL = v
	}
	if v := getEnv("NOTIFICATIONS_PING_THRESHOLD"); v != "" {
		if threshold, err := strconv.ParseFloat(v, 64); err == nil {
			c.Notifications.PingThreshold = threshold
		}
	}
	if v := getEnv("NOTIFICATIONS_UPLOAD_THRESHOLD"); v != "" {
		if threshold, err := strconv.ParseFloat(v, 64); err == nil {
			c.Notifications.UploadThreshold = threshold
		}
	}
	if v := getEnv("NOTIFICATIONS_DOWNLOAD_THRESHOLD"); v != "" {
		if threshold, err := strconv.ParseFloat(v, 64); err == nil {
			c.Notifications.DownloadThreshold = threshold
		}
	}
	if v := getEnv("NOTIFICATIONS_DISCORD_MENTION_ID"); v != "" {
		c.Notifications.DiscordMentionID = v
	}
}

func (c *Config) loadPacketLossFromEnv() {
	if v := getEnv("PACKETLOSS_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.PacketLoss.Enabled = enabled
		}
	}
	if v := getEnv("PACKETLOSS_DEFAULT_INTERVAL"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil {
			c.PacketLoss.DefaultInterval = interval
		}
	}
	if v := getEnv("PACKETLOSS_DEFAULT_PACKET_COUNT"); v != "" {
		if count, err := strconv.Atoi(v); err == nil {
			c.PacketLoss.DefaultPacketCount = count
		}
	}
	if v := getEnv("PACKETLOSS_MAX_CONCURRENT_MONITORS"); v != "" {
		if max, err := strconv.Atoi(v); err == nil {
			c.PacketLoss.MaxConcurrentMonitors = max
		}
	}
	if v := getEnv("PACKETLOSS_PRIVILEGED_MODE"); v != "" {
		if privileged, err := strconv.ParseBool(v); err == nil {
			c.PacketLoss.PrivilegedMode = privileged
		}
	}
	if v := getEnv("PACKETLOSS_RESTORE_MONITORS_ON_STARTUP"); v != "" {
		if restore, err := strconv.ParseBool(v); err == nil {
			c.PacketLoss.RestoreMonitorsOnStartup = restore
		}
	}
}

func (c *Config) loadAgentFromEnv() {
	if v := getEnv("AGENT_HOST"); v != "" {
		c.Agent.Host = v
	}
	if v := getEnv("AGENT_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Agent.Port = port
		}
	}
	if v := getEnv("AGENT_INTERFACE"); v != "" {
		c.Agent.Interface = v
	}
	if v := getEnv("AGENT_API_KEY"); v != "" {
		c.Agent.APIKey = v
	}
	if v := getEnv("AGENT_DISK_INCLUDES"); v != "" {
		c.Agent.DiskIncludes = strings.Split(v, ",")
		for i := range c.Agent.DiskIncludes {
			c.Agent.DiskIncludes[i] = strings.TrimSpace(c.Agent.DiskIncludes[i])
		}
	}
	if v := getEnv("AGENT_DISK_EXCLUDES"); v != "" {
		c.Agent.DiskExcludes = strings.Split(v, ",")
		for i := range c.Agent.DiskExcludes {
			c.Agent.DiskExcludes[i] = strings.TrimSpace(c.Agent.DiskExcludes[i])
		}
	}
}

func (c *Config) loadMonitorFromEnv() {
	if v := getEnv("MONITOR_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.Monitor.Enabled = enabled
		}
	}
	if v := getEnv("MONITOR_RECONNECT_INTERVAL"); v != "" {
		c.Monitor.ReconnectInterval = v
	}
}

func (c *Config) loadTailscaleFromEnv() {
	c.Tailscale.loadFromEnv()
}

func (c *Config) WriteToml(w io.Writer) error {
	cfg := New()
	cfg.Database.Path = "netronome.db"

	secret, err := utils.GenerateSecureToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate session secret: %w", err)
	}
	cfg.Session.Secret = secret

	if isRunningInContainer() {
		cfg.Server.Host = "0.0.0.0"
	}

	if _, err := fmt.Fprintln(w, "# Netronome Configuration"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Database section
	if _, err := fmt.Fprintln(w, "[database]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "type = \"%s\"\n", cfg.Database.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "path = \"%s\"\n", cfg.Database.Path); err != nil {
		return err
	}
	// Postgres options (commented out)
	if _, err := fmt.Fprintln(w, "# PostgreSQL options (uncomment and modify if using postgres)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#host = \"%s\"\n", cfg.Database.Host); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#port = %d\n", cfg.Database.Port); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#user = \"%s\"\n", cfg.Database.User); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#password = \"%s\"\n", cfg.Database.Password); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#dbname = \"%s\"\n", cfg.Database.DBName); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#sslmode = \"%s\"\n\n", cfg.Database.SSLMode); err != nil {
		return err
	}

	// Server section
	if _, err := fmt.Fprintln(w, "[server]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "host = \"%s\"\n", cfg.Server.Host); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "port = %d\n", cfg.Server.Port); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "base_url = \"%s\"\n", cfg.Server.BaseURL); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# gin_mode = \"release\"  # optional: \"debug\" or \"release\""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Logging section
	if _, err := fmt.Fprintln(w, "[logging]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "level = \"%s\"  # trace, debug, info, warn, error, fatal, panic\n", cfg.Logging.Level); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Auth section
	if _, err := fmt.Fprintln(w, "[auth]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Whitelist specific networks to bypass authentication, using CIDR notation."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Example: whitelist = [\"127.0.0.1/32\"]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "whitelist = []"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// OIDC section
	if _, err := fmt.Fprintln(w, "#[oidc]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#issuer = \"%s\"\n", cfg.OIDC.Issuer); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#client_id = \"%s\"\n", cfg.OIDC.ClientID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#client_secret = \"%s\"\n", cfg.OIDC.ClientSecret); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#redirect_url = \"%s\"\n", cfg.OIDC.RedirectURL); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// SpeedTest section
	if _, err := fmt.Fprintln(w, "[speedtest]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "timeout = %d\n", cfg.SpeedTest.Timeout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// SpeedTest IPerf section
	if _, err := fmt.Fprintln(w, "[speedtest.iperf]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "test_duration = %d\n", cfg.SpeedTest.IPerf.TestDuration); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "parallel_conns = %d\n", cfg.SpeedTest.IPerf.ParallelConns); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "timeout = %d\n", cfg.SpeedTest.IPerf.Timeout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// SpeedTest Librespeed section
	if _, err := fmt.Fprintln(w, "[speedtest.librespeed]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "timeout = %d\n", cfg.SpeedTest.Librespeed.Timeout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// SpeedTest IPerf Ping section
	if _, err := fmt.Fprintln(w, "[speedtest.iperf.ping]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "count = %d\n", cfg.SpeedTest.IPerf.Ping.Count); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "interval = %d\n", cfg.SpeedTest.IPerf.Ping.Interval); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "timeout = %d\n", cfg.SpeedTest.IPerf.Ping.Timeout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Pagination section (commented out)
	if _, err := fmt.Fprintln(w, "# Pagination options (defaults work well for most cases)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Only uncomment and modify if you need to adjust the API response pagination"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "#[pagination]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#default_page = %d\n", cfg.Pagination.DefaultPage); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#default_page_size = %d\n", cfg.Pagination.DefaultPageSize); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#max_page_size = %d\n", cfg.Pagination.MaxPageSize); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#default_time_range = \"%s\"\n", cfg.Pagination.DefaultTimeRange); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#default_limit = %d\n", cfg.Pagination.DefaultLimit); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// Session section
	if _, err := fmt.Fprintln(w, "[session]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "session_secret = \"%s\"\n", cfg.Session.Secret); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}

	// GeoIP section (commented out by default)
	if _, err := fmt.Fprintln(w, "# GeoIP configuration for country flags and ASN info in traceroute"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Uncomment and configure database paths to enable. See README for setup instructions."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "#[geoip]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#country_database_path = \"/path/to/GeoLite2-Country.mmdb\"\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "#asn_database_path = \"/path/to/GeoLite2-ASN.mmdb\"\n"); err != nil {
		return err
	}

	// Notifications section
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[notifications]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "enabled = %v\n", cfg.Notifications.Enabled); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "webhook_url = \"%s\"\n", cfg.Notifications.WebhookURL); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "ping_threshold = %v\n", cfg.Notifications.PingThreshold); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "upload_threshold = %v\n", cfg.Notifications.UploadThreshold); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "download_threshold = %v\n", cfg.Notifications.DownloadThreshold); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "discord_mention_id = \"%s\"\n", cfg.Notifications.DiscordMentionID); err != nil {
		return err
	}

	// Packet Loss section
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[packetloss]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "enabled = %v\n", cfg.PacketLoss.Enabled); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "default_interval = %d # seconds between tests\n", cfg.PacketLoss.DefaultInterval); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "default_packet_count = %d # packets per test\n", cfg.PacketLoss.DefaultPacketCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "max_concurrent_monitors = %d\n", cfg.PacketLoss.MaxConcurrentMonitors); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "privileged_mode = %v # Use privileged ICMP mode for better MTR support (requires root/sudo)\n", cfg.PacketLoss.PrivilegedMode); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "restore_monitors_on_startup = %v # WARNING: If true, immediately runs ALL enabled packet loss monitors on startup (may cause network congestion). Default: monitors run on their scheduled intervals only\n", cfg.PacketLoss.RestoreMonitorsOnStartup); err != nil {
		return err
	}

	// Agent section
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[agent]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "host = \"%s\" # IP address to bind to (0.0.0.0 for all interfaces)\n", cfg.Agent.Host); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "port = %d\n", cfg.Agent.Port); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "interface = \"%s\" # empty for all interfaces\n", cfg.Agent.Interface); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# api_key = \"\" # API key for agent authentication (optional)"); err != nil {
		return err
	}

	// Monitor section
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[monitor]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "enabled = %v\n", cfg.Monitor.Enabled); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "reconnect_interval = \"%s\"\n", cfg.Monitor.ReconnectInterval); err != nil {
		return err
	}

	// Tailscale section
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Tailscale integration for secure agent-to-server communication"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "[tailscale]"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "enabled = %v\n", cfg.Tailscale.Enabled); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "method = \"%s\" # \"auto\" (default), \"host\", or \"tsnet\"\n", cfg.Tailscale.Method); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TSNet settings (used when method=\"tsnet\" or auto-detected)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# auth_key = \"\" # Required for tsnet mode"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# hostname = \"\" # Custom hostname (optional)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# ephemeral = %v # Remove on shutdown\n", cfg.Tailscale.Ephemeral); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# state_dir = \"%s\" # Directory for tsnet state\n", cfg.Tailscale.StateDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# control_url = \"\" # For Headscale (optional)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Agent settings"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "agent_port = %d # Port for agent to listen on (0 = use agent's port)\n", cfg.Tailscale.AgentPort); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Server discovery settings"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "auto_discover = %v # Auto-discover Tailscale agents\n", cfg.Tailscale.AutoDiscover); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "discovery_interval = \"%s\" # How often to check for new agents\n", cfg.Tailscale.DiscoveryInterval); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "discovery_port = %d # Port to probe for agents\n", cfg.Tailscale.DiscoveryPort); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "# discovery_prefix = \"\" # Only discover agents with this prefix (optional)\n"); err != nil {
		return err
	}

	return nil
}

func GetDefaultConfigPath() string {
	if configDir, err := os.UserConfigDir(); err == nil {
		configPath := filepath.Join(configDir, AppName, "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	return "config.toml"
}

func DefaultConfigPaths() []string {
	var paths []string

	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", AppName, "config.toml"))
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(configDir, AppName, "config.toml"))
	}

	// Finally try current directory
	paths = append(paths, "config.toml")

	return paths
}

// EnsureConfig ensures a config file exists at the given path or in default locations,
// generating one if necessary. If configPath is empty, it will check default locations.
func EnsureConfig(configPath string) (string, error) {
	if configPath != "" {
		// if specific config path provided, create it if it doesn't exist
		if _, err := os.Stat(configPath); err != nil {
			// create the directory structure if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				return "", fmt.Errorf("failed to create config directory: %w", err)
			}

			cfg := New()
			f, err := os.Create(configPath)
			if err != nil {
				return "", fmt.Errorf("failed to create config file: %w", err)
			}
			defer f.Close()

			if err := cfg.WriteToml(f); err != nil {
				return "", fmt.Errorf("failed to write config file: %w", err)
			}

			log.Info().Str("path", configPath).Msg("Generated default config file")
		}
		return configPath, nil
	}

	paths := DefaultConfigPaths()
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".config")
		netronomeDir := filepath.Join(configDir, AppName)
		if err := os.MkdirAll(netronomeDir, 0755); err == nil {
			configPath = filepath.Join(netronomeDir, "config.toml")
		} else {
			// fall back to platform-specific user config dir
			if configDir, err := os.UserConfigDir(); err == nil {
				netronomeDir := filepath.Join(configDir, AppName)
				if err := os.MkdirAll(netronomeDir, 0755); err == nil {
					configPath = filepath.Join(netronomeDir, "config.toml")
				} else {
					log.Warn().
						Err(err).
						Msg("could not create config directory, falling back to working directory")
					configPath = "config.toml"
				}
			} else {
				log.Warn().
					Err(err).
					Msg("could not determine user config directory, falling back to working directory")
				configPath = "config.toml"
			}
		}
	}

	cfg := New()
	f, err := os.Create(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	if err := cfg.WriteToml(f); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	log.Info().Str("path", configPath).Msg("Generated default config file")
	return configPath, nil
}

func getEnv(key string) string {
	return os.Getenv(EnvPrefix + key)
}

// GetEffectiveMethod returns the effective Tailscale method based on configuration
func (t *TailscaleConfig) GetEffectiveMethod() (string, error) {
	if !t.Enabled {
		return "", nil
	}

	switch t.Method {
	case "auto":
		// Auto-detect based on auth key presence
		if t.AuthKey != "" {
			return "tsnet", nil
		}
		return "host", nil
	case "host":
		return "host", nil
	case "tsnet":
		// TSNet requires auth key
		if t.AuthKey == "" {
			return "", fmt.Errorf("auth_key is required when method is 'tsnet'")
		}
		return "tsnet", nil
	default:
		return "", fmt.Errorf("invalid method: %s", t.Method)
	}
}

// Validate checks if the Tailscale configuration is valid
func (t *TailscaleConfig) Validate() error {
	if !t.Enabled {
		return nil
	}

	// Validate method
	if t.Method != "auto" && t.Method != "host" && t.Method != "tsnet" {
		return fmt.Errorf("invalid method: %s", t.Method)
	}

	// Check if tsnet requires auth key
	method, err := t.GetEffectiveMethod()
	if err != nil {
		return err
	}
	_ = method

	// Validate ports
	if t.AgentPort < 0 || t.AgentPort > 65535 {
		return fmt.Errorf("invalid agent port: %d", t.AgentPort)
	}

	if t.DiscoveryPort < 0 || t.DiscoveryPort > 65535 {
		return fmt.Errorf("invalid discovery port: %d", t.DiscoveryPort)
	}

	// Validate discovery interval if auto-discover is enabled
	if t.AutoDiscover && t.DiscoveryInterval != "" {
		if _, err := time.ParseDuration(t.DiscoveryInterval); err != nil {
			return fmt.Errorf("invalid discovery interval: %s", t.DiscoveryInterval)
		}
	}

	return nil
}

// MigrateFromOldFormat migrates from the old dual-section format to the new unified format
func (t *TailscaleConfig) MigrateFromOldFormat() TailscaleConfig {
	result := *t // Copy current config

	// If Method is not set, determine from old fields
	if result.Method == "" {
		if t.PreferHost {
			result.Method = "host"
		} else if t.AuthKey != "" {
			result.Method = "tsnet"
		} else {
			result.Method = "host" // Default to host
		}
	}

	// Migrate agent settings
	if t.Agent.Enabled && result.AgentPort == 0 {
		result.AgentPort = t.Agent.Port
	}

	// Migrate monitor settings
	if result.AutoDiscover == false && t.Monitor.AutoDiscover {
		result.AutoDiscover = t.Monitor.AutoDiscover
	}
	if result.DiscoveryInterval == "" && t.Monitor.DiscoveryInterval != "" {
		result.DiscoveryInterval = t.Monitor.DiscoveryInterval
	}
	if result.DiscoveryPort == 0 && t.Monitor.DiscoveryPort != 0 {
		result.DiscoveryPort = t.Monitor.DiscoveryPort
	}
	if result.DiscoveryPrefix == "" && t.Monitor.DiscoveryPrefix != "" {
		result.DiscoveryPrefix = t.Monitor.DiscoveryPrefix
	}

	return result
}

// IsAgentMode returns true if Tailscale is enabled for agent connectivity
func (t *TailscaleConfig) IsAgentMode() bool {
	if !t.Enabled {
		return false
	}

	method, err := t.GetEffectiveMethod()
	if err != nil {
		return false
	}

	return method == "host" || method == "tsnet"
}

// IsServerDiscoveryMode returns true if Tailscale server discovery is enabled
func (t *TailscaleConfig) IsServerDiscoveryMode() bool {
	return t.Enabled && t.AutoDiscover
}

// loadFromEnv is a method to load environment variables for TailscaleConfig
func (t *TailscaleConfig) loadFromEnv() {
	// Core settings
	if v := getEnv("TAILSCALE_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			t.Enabled = enabled
		}
	}
	if v := getEnv("TAILSCALE_METHOD"); v != "" {
		t.Method = v
	}
	// Always handle AUTH_KEY even if empty to allow clearing
	if v, exists := os.LookupEnv(EnvPrefix + "TAILSCALE_AUTH_KEY"); exists {
		t.AuthKey = v
	}
	
	// TSNet settings
	if v := getEnv("TAILSCALE_HOSTNAME"); v != "" {
		t.Hostname = v
	}
	if v := getEnv("TAILSCALE_EPHEMERAL"); v != "" {
		if ephemeral, err := strconv.ParseBool(v); err == nil {
			t.Ephemeral = ephemeral
		}
	}
	if v := getEnv("TAILSCALE_STATE_DIR"); v != "" {
		t.StateDir = v
	}
	if v := getEnv("TAILSCALE_CONTROL_URL"); v != "" {
		t.ControlURL = v
	}
	
	// Agent settings
	if v := getEnv("TAILSCALE_AGENT_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			t.AgentPort = port
		}
	}
	
	// Server discovery settings
	if v := getEnv("TAILSCALE_AUTO_DISCOVER"); v != "" {
		if auto, err := strconv.ParseBool(v); err == nil {
			t.AutoDiscover = auto
		}
	}
	if v := getEnv("TAILSCALE_DISCOVERY_INTERVAL"); v != "" {
		t.DiscoveryInterval = v
	}
	if v := getEnv("TAILSCALE_DISCOVERY_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			t.DiscoveryPort = port
		}
	}
	if v := getEnv("TAILSCALE_DISCOVERY_PREFIX"); v != "" {
		t.DiscoveryPrefix = v
	}
	
	// Deprecated fields - still support for backward compatibility
	if v := getEnv("TAILSCALE_PREFER_HOST"); v != "" {
		if preferHost, err := strconv.ParseBool(v); err == nil {
			t.PreferHost = preferHost
			// If prefer_host is true and method wasn't explicitly set, use "host"
			if preferHost && getEnv("TAILSCALE_METHOD") == "" {
				t.Method = "host"
			}
		}
	}
	if v := getEnv("TAILSCALE_AGENT_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			t.Agent.Enabled = enabled
		}
	}
	if v := getEnv("TAILSCALE_AGENT_ACCEPT_ROUTES"); v != "" {
		if accept, err := strconv.ParseBool(v); err == nil {
			t.Agent.AcceptRoutes = accept
		}
	}
	if v := getEnv("TAILSCALE_AGENT_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			t.Agent.Port = port
			// Also set new field if not already set
			if t.AgentPort == 0 {
				t.AgentPort = port
			}
		}
	}
	if v := getEnv("TAILSCALE_MONITOR_AUTO_DISCOVER"); v != "" {
		if auto, err := strconv.ParseBool(v); err == nil {
			t.Monitor.AutoDiscover = auto
			// Also set new field
			t.AutoDiscover = auto
		}
	}
	if v := getEnv("TAILSCALE_MONITOR_DISCOVERY_INTERVAL"); v != "" {
		t.Monitor.DiscoveryInterval = v
		// Also set new field
		t.DiscoveryInterval = v
	}
	if v := getEnv("TAILSCALE_MONITOR_DISCOVERY_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			t.Monitor.DiscoveryPort = port
			// Also set new field
			t.DiscoveryPort = port
		}
	}
	if v := getEnv("TAILSCALE_MONITOR_DISCOVERY_PREFIX"); v != "" {
		t.Monitor.DiscoveryPrefix = v
		// Also set new field
		t.DiscoveryPrefix = v
	}
}
