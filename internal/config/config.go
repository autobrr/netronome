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

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog/log"
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
	Database   DatabaseConfig   `toml:"database"`
	Server     ServerConfig     `toml:"server"`
	Logging    LoggingConfig    `toml:"logging"`
	OIDC       OIDCConfig       `toml:"oidc"`
	SpeedTest  SpeedTestConfig  `toml:"speedtest"`
	Pagination PaginationConfig `toml:"pagination"`
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

type OIDCConfig struct {
	Issuer       string `toml:"issuer" env:"OIDC_ISSUER"`
	ClientID     string `toml:"client_id" env:"OIDC_CLIENT_ID"`
	ClientSecret string `toml:"client_secret" env:"OIDC_CLIENT_SECRET"`
	RedirectURL  string `toml:"redirect_url" env:"OIDC_REDIRECT_URL"`
}

type SpeedTestConfig struct {
	IPerf   IperfConfig `toml:"iperf"`
	Timeout int         `toml:"timeout" env:"SPEEDTEST_TIMEOUT"` // Timeout in seconds
}

type IperfConfig struct {
	TestDuration  int `toml:"test_duration" env:"IPERF_TEST_DURATION"`
	ParallelConns int `toml:"parallel_conns" env:"IPERF_PARALLEL_CONNS"`
}

type PaginationConfig struct {
	DefaultPage      int    `toml:"default_page" env:"DEFAULT_PAGE"`
	DefaultPageSize  int    `toml:"default_page_size" env:"DEFAULT_PAGE_SIZE"`
	MaxPageSize      int    `toml:"max_page_size" env:"MAX_PAGE_SIZE"`
	DefaultTimeRange string `toml:"default_time_range" env:"DEFAULT_TIME_RANGE"`
	DefaultLimit     int    `toml:"default_limit" env:"DEFAULT_LIMIT"`
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
			Host: "127.0.0.1",
			Port: 7575,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		SpeedTest: SpeedTestConfig{
			IPerf: IperfConfig{
				TestDuration:  10,
				ParallelConns: 4,
			},
			Timeout: 30, // 30 seconds default timeout
		},
		Pagination: PaginationConfig{
			DefaultPage:      1,
			DefaultPageSize:  20,
			MaxPageSize:      100,
			DefaultTimeRange: "1w",
			DefaultLimit:     20,
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
		log.Info().
			Str("path", configPath).
			Msg("Loaded configuration file")

		// If db path is relative, make it relative to config file
		if !filepath.IsAbs(cfg.Database.Path) {
			cfg.Database.Path = filepath.Join(filepath.Dir(configPath), cfg.Database.Path)
		}
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
					log.Info().
						Str("path", path).
						Msg("Loaded configuration file")
					found = true

					// If db path is relative, make it relative to config file
					if !filepath.IsAbs(cfg.Database.Path) {
						cfg.Database.Path = filepath.Join(filepath.Dir(path), cfg.Database.Path)
					}
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
	// Database
	c.loadDatabaseFromEnv()

	// Server
	c.loadServerFromEnv()

	// Logging
	c.loadLoggingFromEnv()

	// OIDC
	c.loadOIDCFromEnv()

	// SpeedTest
	c.loadSpeedTestFromEnv()

	// Pagination
	c.loadPaginationFromEnv()

	return nil
}

func (c *Config) loadDatabaseFromEnv() {
	if v := os.Getenv(EnvPrefix + "DB_TYPE"); v != "" {
		c.Database.Type = DatabaseType(v)
	}
	if v := os.Getenv(EnvPrefix + "DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv(EnvPrefix + "DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv(EnvPrefix + "DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv(EnvPrefix + "DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv(EnvPrefix + "DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv(EnvPrefix + "DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}
	if v := os.Getenv(EnvPrefix + "DB_PATH"); v != "" {
		c.Database.Path = v
	}
}

func (c *Config) loadServerFromEnv() {
	if v := os.Getenv(EnvPrefix + "HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv(EnvPrefix + "PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv(EnvPrefix + "BASE_URL"); v != "" {
		c.Server.BaseURL = v
	}
	if v := os.Getenv(EnvPrefix + "GIN_MODE"); v != "" {
		c.Server.GinMode = v
	}
}

func (c *Config) loadLoggingFromEnv() {
	if v := os.Getenv(EnvPrefix + "LOG_LEVEL"); v != "" {
		c.Logging.Level = strings.ToLower(v)
	}
}

func (c *Config) loadOIDCFromEnv() {
	if v := os.Getenv(EnvPrefix + "OIDC_ISSUER"); v != "" {
		c.OIDC.Issuer = v
	}
	if v := os.Getenv(EnvPrefix + "OIDC_CLIENT_ID"); v != "" {
		c.OIDC.ClientID = v
	}
	if v := os.Getenv(EnvPrefix + "OIDC_CLIENT_SECRET"); v != "" {
		c.OIDC.ClientSecret = v
	}
	if v := os.Getenv(EnvPrefix + "OIDC_REDIRECT_URL"); v != "" {
		c.OIDC.RedirectURL = v
	}
}

func (c *Config) loadSpeedTestFromEnv() {
	if v := os.Getenv(EnvPrefix + "IPERF_TEST_DURATION"); v != "" {
		if duration, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.TestDuration = duration
		}
	}
	if v := os.Getenv(EnvPrefix + "IPERF_PARALLEL_CONNS"); v != "" {
		if conns, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.IPerf.ParallelConns = conns
		}
	}
	if v := os.Getenv(EnvPrefix + "SPEEDTEST_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.SpeedTest.Timeout = timeout
		}
	}
}

func (c *Config) loadPaginationFromEnv() {
	if v := os.Getenv(EnvPrefix + "DEFAULT_PAGE"); v != "" {
		if page, err := strconv.Atoi(v); err == nil {
			c.Pagination.DefaultPage = page
		}
	}
	if v := os.Getenv(EnvPrefix + "DEFAULT_PAGE_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			c.Pagination.DefaultPageSize = size
		}
	}
	if v := os.Getenv(EnvPrefix + "MAX_PAGE_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			c.Pagination.MaxPageSize = size
		}
	}
	if v := os.Getenv(EnvPrefix + "DEFAULT_TIME_RANGE"); v != "" {
		c.Pagination.DefaultTimeRange = v
	}
	if v := os.Getenv(EnvPrefix + "DEFAULT_LIMIT"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil {
			c.Pagination.DefaultLimit = limit
		}
	}
}

func (c *Config) WriteToml(w io.Writer) error {
	// Create a copy of config with default values
	cfg := New()
	cfg.Database.Path = "netronome.db"

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

	return nil
}

func GetDefaultConfigPath() string {
	// Check user config directory first (~/.config/netronome/config.toml on Linux)
	if configDir, err := os.UserConfigDir(); err == nil {
		configPath := filepath.Join(configDir, AppName, "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// Fallback to current directory
	return "config.toml"
}

// DefaultConfigPaths returns all possible config file locations in order of preference
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

			// generate the config file
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

	// try each default path
	paths := DefaultConfigPaths()
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// no config found, generate one in the default location
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

	// generate the config file
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
