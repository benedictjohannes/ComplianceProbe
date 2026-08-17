package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

// GetFileSystem returns an http.FileSystem serving the embedded dist directory.
func GetFileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// GetIndexHTML returns the raw bytes of the embedded index.html.
func GetIndexHTML() ([]byte, error) {
	return distFS.ReadFile("dist/index.html")
}
