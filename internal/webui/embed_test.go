package webui

import (
	"testing"
)

func TestGetFileSystem(t *testing.T) {
	fs, err := GetFileSystem()
	if err != nil {
		t.Fatalf("GetFileSystem() returned error: %v", err)
	}
	if fs == nil {
		t.Fatal("GetFileSystem() returned nil fs")
	}

	// Verify we can open index.html from http.FileSystem
	f, err := fs.Open("/index.html")
	if err != nil {
		t.Fatalf("fs.Open('/index.html') returned error: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("f.Stat() returned error: %v", err)
	}
	if stat.Size() == 0 {
		t.Errorf("expected non-empty index.html, got size %d", stat.Size())
	}
}

func TestGetIndexHTML(t *testing.T) {
	data, err := GetIndexHTML()
	if err != nil {
		t.Fatalf("GetIndexHTML() returned error: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("GetIndexHTML() returned empty byte slice")
	}
}
