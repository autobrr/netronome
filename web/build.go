// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package web

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

//go:embed all:dist
var Dist embed.FS

var DistDirFS = MustSubFS(Dist, "dist")

// BuildFrontend executes the frontend build process
func BuildFrontend() error {
	webDir, err := filepath.Abs("web")
	if err != nil {
		return fmt.Errorf("failed to get web directory path: %w", err)
	}

	log.Info().Str("webDir", webDir).Msg("Building frontend in directory")

	// Run pnpm install
	installCmd := exec.Command("pnpm", "install")
	installCmd.Dir = webDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to run pnpm install: %w", err)
	}

	// Run pnpm build
	buildCmd := exec.Command("pnpm", "build")
	buildCmd.Dir = webDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to run pnpm build: %w", err)
	}

	// Verify dist directory was created
	distDir := filepath.Join(webDir, "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		return fmt.Errorf("dist directory was not created at %s", distDir)
	}

	log.Info().Str("distDir", distDir).Msg("Frontend built successfully")
	return nil
}

// MustSubFS creates sub FS from current filesystem or panic on failure
func MustSubFS(currentFs fs.FS, fsRoot string) fs.FS {
	subFs, err := fs.Sub(currentFs, fsRoot)
	if err != nil {
		panic(fmt.Errorf("cannot create sub FS: %w", err))
	}
	return subFs
}

// ServeStatic registers static file handlers with Gin
func ServeStatic(r *gin.Engine) {
	// Add no-cache headers for index.html
	r.Use(func(c *gin.Context) {
		if strings.HasSuffix(c.Request.URL.Path, ".html") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	// Handle all non-API routes with index.html
	r.NoRoute(func(c *gin.Context) {
		baseURL := c.GetString("base_url")
		if baseURL == "" {
			baseURL = "/"
		}

		// Don't handle API routes
		if strings.HasPrefix(c.Request.URL.Path, strings.TrimSuffix(baseURL, "/")+"/api") {
			c.AbortWithStatus(404)
			return
		}

		// For root base URL, register the root handlers
		if baseURL == "/" {
			switch c.Request.URL.Path {
			case "/":
				ServeIndex(c)
				return
			case "/index.html":
				ServeIndex(c)
				return
			case "/favicon.ico":
				ServeStaticFile(c)
				return
			}
			if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
				ServeStaticFile(c)
				return
			}
		}

		// Redirect paths without trailing slash to paths with trailing slash
		// Only for non-root base URL paths
		if baseURL != "/" && c.Request.URL.Path == strings.TrimSuffix(baseURL, "/") {
			c.Redirect(http.StatusMovedPermanently, baseURL)
			return
		}

		ServeIndex(c)
	})
}

// serveStaticFile serves static files with proper headers
func serveStaticFile(c *gin.Context) {
	filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
	serveFileFromFS(c, filePath)
}

// serveFileFromFS serves a file from the embedded filesystem with proper headers
func serveFileFromFS(c *gin.Context, filepath string) {
	file, err := DistDirFS.Open(filepath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Set content type based on file extension
	ext := strings.ToLower(path.Ext(filepath))
	var contentType string
	switch ext {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "text/javascript; charset=utf-8"
	case ".svg":
		contentType = "image/svg+xml"
	case ".ico":
		contentType = "image/x-icon"
	case ".png":
		contentType = "image/png"
	default:
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000")
	http.ServeContent(c.Writer, c.Request, filepath, stat.ModTime(), file.(io.ReadSeeker))
}

// ServeIndex serves index.html with proper headers
func ServeIndex(c *gin.Context) {
	file, err := DistDirFS.Open("index.html")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Get base URL from gin context
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "/"
	}

	// Replace the template variable
	html := strings.Replace(string(content), "{{.BaseURL}}", baseURL, 1)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, html)
}

// ServeStaticFile serves static files with proper headers
func ServeStaticFile(c *gin.Context) {
	filePath := strings.TrimPrefix(c.Request.URL.Path, c.GetString("base_url"))
	filePath = strings.TrimPrefix(filePath, "/")
	serveFileFromFS(c, filePath)
}

// Verify dist directory exists in embedded FS
func init() {
	if _, err := Dist.ReadDir("dist"); err != nil {
		panic(fmt.Errorf("dist directory not found in embedded filesystem: %w", err))
	}
}
