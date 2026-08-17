package server

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/benedictjohannes/crobe/report"
	"github.com/klauspost/compress/zstd"
)

func TestBuildReportArchive(t *testing.T) {
	dummyRes := report.FinalResult{
		Structured: report.FinalReport{
			OS:         "linux",
			Stats:      report.Stats{Passed: 2, Failed: 0},
			Assertions: make(map[string]report.Assertion),
		},
		Markdown: "# Test Report\nEverything passed.",
		Log:      "=== REPORT LOG ===\nAll clear.",
	}

	// 1. Default format (defaults to zip)
	data, mimeType, filename, err := BuildReportArchive(dummyRes, "")
	if err != nil {
		t.Fatalf("BuildReportArchive with empty format failed: %v", err)
	}
	if mimeType != "application/zip" || filename != "report.zip" {
		t.Errorf("expected application/zip and report.zip, got %s and %s", mimeType, filename)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) != 3 {
		t.Fatalf("zip reader failed or file count mismatch: %v, count=%d", err, len(zr.File))
	}

	// 2. Explicit zip format
	data, mimeType, filename, err = BuildReportArchive(dummyRes, "zip")
	if err != nil {
		t.Fatalf("BuildReportArchive(zip) failed: %v", err)
	}
	if mimeType != "application/zip" || filename != "report.zip" {
		t.Errorf("expected application/zip and report.zip, got %s and %s", mimeType, filename)
	}

	// 3. Tar format
	data, mimeType, filename, err = BuildReportArchive(dummyRes, "tar")
	if err != nil {
		t.Fatalf("BuildReportArchive(tar) failed: %v", err)
	}
	if mimeType != "application/x-tar" || filename != "report.tar" {
		t.Errorf("expected application/x-tar and report.tar, got %s and %s", mimeType, filename)
	}
	tr := tar.NewReader(bytes.NewReader(data))
	fileCount := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		if hdr.Name != "report.json" && hdr.Name != "report.md" && hdr.Name != "report.log" {
			t.Errorf("unexpected file in tar: %s", hdr.Name)
		}
		fileCount++
	}
	if fileCount != 3 {
		t.Errorf("expected 3 files in tar, got %d", fileCount)
	}

	// 4. Tar.gz / tgz format
	for _, fmtStr := range []string{"tar.gz", "tgz"} {
		data, mimeType, filename, err = BuildReportArchive(dummyRes, fmtStr)
		if err != nil {
			t.Fatalf("BuildReportArchive(%s) failed: %v", fmtStr, err)
		}
		if mimeType != "application/gzip" || filename != "report.tar.gz" {
			t.Errorf("expected application/gzip and report.tar.gz, got %s and %s", mimeType, filename)
		}
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("gzip reader failed: %v", err)
		}
		tr = tar.NewReader(gr)
		fileCount = 0
		for {
			_, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("tar.gz read error: %v", err)
			}
			fileCount++
		}
		if fileCount != 3 {
			t.Errorf("expected 3 files in tar.gz, got %d", fileCount)
		}
	}

	// 5. Tar.zst / tzst / zst format
	for _, fmtStr := range []string{"tar.zst", "tzst", "zst"} {
		data, mimeType, filename, err = BuildReportArchive(dummyRes, fmtStr)
		if err != nil {
			t.Fatalf("BuildReportArchive(%s) failed: %v", fmtStr, err)
		}
		if mimeType != "application/zstd" || filename != "report.tar.zst" {
			t.Errorf("expected application/zstd and report.tar.zst, got %s and %s", mimeType, filename)
		}
		zr, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("zstd reader failed: %v", err)
		}
		tr = tar.NewReader(zr)
		fileCount = 0
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("tar.zst read error: %v", err)
			}
			if hdr.Name != "report.json" && hdr.Name != "report.md" && hdr.Name != "report.log" {
				t.Errorf("unexpected file in tar.zst: %s", hdr.Name)
			}
			fileCount++
		}
		zr.Close()
		if fileCount != 3 {
			t.Errorf("expected 3 files in tar.zst, got %d", fileCount)
		}
	}

	// 6. Unsupported format returns error
	_, _, _, err = BuildReportArchive(dummyRes, "invalid_fmt")
	if err == nil {
		t.Errorf("expected error for invalid archive format, got nil")
	}
}
