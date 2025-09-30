package dashboard

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// Static files for the dashboard web app
// These files are copied during build process
//go:embed static/*
var dashboardAssets embed.FS

// GetFileSystem returns the embedded dashboard file system
func GetFileSystem() (fs.FS, error) {
	return fs.Sub(dashboardAssets, "static")
}

// GetHandler returns an HTTP handler for serving the dashboard
func GetHandler() (http.Handler, error) {
	dashboardFS, err := GetFileSystem()
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path - now serving from root
		path := r.URL.Path
		if path == "" || path == "/" {
			path = "index.html"
		} else {
			// Remove leading slash for embedded filesystem
			path = strings.TrimPrefix(path, "/")
		}

		// Check if the file exists
		file, err := dashboardFS.Open(path)
		if err != nil {
			// If file doesn't exist, serve index.html for SPA routing
			path = "index.html"
			file, err = dashboardFS.Open(path)
			if err != nil {
				http.Error(w, "Dashboard not found", http.StatusNotFound)
				return
			}
		}
		defer file.Close()

		// Get file info for content length
		stat, err := file.Stat()
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		// Set the correct content type for specific files
		ext := filepath.Ext(path)
		switch ext {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		// Set caching headers for static assets
		if ext == ".js" || ext == ".css" || ext == ".ico" {
			w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		// Set content length
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

		// Copy file content to response
		content, err := fs.ReadFile(dashboardFS, path)
		if err != nil {
			http.Error(w, "Error reading file content", http.StatusInternalServerError)
			return
		}

		w.Write(content)
	}), nil
}