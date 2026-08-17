package server

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"time"

	"github.com/benedictjohannes/crobe/report"
	"github.com/klauspost/compress/zstd"
)

// BuildReportArchive creates an in-memory archive bundle of report files.
func BuildReportArchive(res report.FinalResult, format string) ([]byte, string, string, error) {
	if format == "" {
		format = "zip"
	}

	jsonBytes, err := json.MarshalIndent(res.Structured, "", "  ")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	files := []struct {
		name string
		data []byte
	}{
		{"report.json", jsonBytes},
		{"report.md", []byte(res.Markdown)},
		{"report.log", []byte(res.Log)},
	}

	switch format {
	case "zip":
		buf := new(bytes.Buffer)
		zw := zip.NewWriter(buf)
		for _, f := range files {
			w, err := zw.Create(f.name)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to create zip entry %s: %w", f.name, err)
			}
			if _, err := w.Write(f.data); err != nil {
				return nil, "", "", fmt.Errorf("failed to write zip entry %s: %w", f.name, err)
			}
		}
		if err := zw.Close(); err != nil {
			return nil, "", "", fmt.Errorf("failed to finalize zip: %w", err)
		}
		return buf.Bytes(), "application/zip", "report.zip", nil

	case "tar":
		buf := new(bytes.Buffer)
		tw := tar.NewWriter(buf)
		now := time.Now()
		for _, f := range files {
			hdr := &tar.Header{
				Name:    f.name,
				Mode:    0644,
				Size:    int64(len(f.data)),
				ModTime: now,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return nil, "", "", fmt.Errorf("failed to write tar header %s: %w", f.name, err)
			}
			if _, err := tw.Write(f.data); err != nil {
				return nil, "", "", fmt.Errorf("failed to write tar entry %s: %w", f.name, err)
			}
		}
		if err := tw.Close(); err != nil {
			return nil, "", "", fmt.Errorf("failed to finalize tar: %w", err)
		}
		return buf.Bytes(), "application/x-tar", "report.tar", nil

	case "tar.gz", "tgz":
		buf := new(bytes.Buffer)
		gw := gzip.NewWriter(buf)
		tw := tar.NewWriter(gw)
		now := time.Now()
		for _, f := range files {
			hdr := &tar.Header{
				Name:    f.name,
				Mode:    0644,
				Size:    int64(len(f.data)),
				ModTime: now,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return nil, "", "", fmt.Errorf("failed to write tar header %s: %w", f.name, err)
			}
			if _, err := tw.Write(f.data); err != nil {
				return nil, "", "", fmt.Errorf("failed to write tar entry %s: %w", f.name, err)
			}
		}
		if err := tw.Close(); err != nil {
			return nil, "", "", fmt.Errorf("failed to finalize tar: %w", err)
		}
		if err := gw.Close(); err != nil {
			return nil, "", "", fmt.Errorf("failed to finalize gzip: %w", err)
		}
		return buf.Bytes(), "application/gzip", "report.tar.gz", nil

	case "tar.zst", "tzst", "zst":
		buf := new(bytes.Buffer)
		zw, err := zstd.NewWriter(buf)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to create zstd writer: %w", err)
		}
		tw := tar.NewWriter(zw)
		now := time.Now()
		for _, f := range files {
			hdr := &tar.Header{
				Name:    f.name,
				Mode:    0644,
				Size:    int64(len(f.data)),
				ModTime: now,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return nil, "", "", fmt.Errorf("failed to write tar header %s: %w", f.name, err)
			}
			if _, err := tw.Write(f.data); err != nil {
				return nil, "", "", fmt.Errorf("failed to write tar entry %s: %w", f.name, err)
			}
		}
		if err := tw.Close(); err != nil {
			return nil, "", "", fmt.Errorf("failed to finalize tar: %w", err)
		}
		if err := zw.Close(); err != nil {
			return nil, "", "", fmt.Errorf("failed to finalize zstd: %w", err)
		}
		return buf.Bytes(), "application/zstd", "report.tar.zst", nil

	default:
		return nil, "", "", fmt.Errorf("unsupported archive format: %s. Supported formats: zip, tar, tar.gz, tar.zst", format)
	}
}
