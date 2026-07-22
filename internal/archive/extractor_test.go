package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// Create test zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create test zip: %v", err)
	}

	w := zip.NewWriter(zipFile)

	// Add file 1
	f1, err := w.Create("hello.txt")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	f1.Write([]byte("hello world"))

	// Add file 2 inside dir
	f2, err := w.Create("src/main.go")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}
	f2.Write([]byte("package main\nfunc main(){}"))

	// Add .git file (should be ignored)
	fGit, err := w.Create(".git/config")
	if err != nil {
		t.Fatalf("Failed to create git zip entry: %v", err)
	}
	fGit.Write([]byte("should be skipped"))

	// Add manifest file
	fManifest, err := w.Create(".tg-git.yaml")
	if err != nil {
		t.Fatalf("Failed to create manifest entry: %v", err)
	}
	fManifest.Write([]byte("repo: git@github.com:test/repo.git\nbranch: feature-test\nmessage: 'Test commit'"))

	w.Close()
	zipFile.Close()

	// Perform extraction
	res, err := ExtractZip(zipPath, destDir)
	if err != nil {
		t.Fatalf("ExtractZip failed: %v", err)
	}

	if res.Manifest == nil {
		t.Errorf("Expected manifest to be parsed, got nil")
	} else {
		if res.Manifest.Repo != "git@github.com:test/repo.git" {
			t.Errorf("Unexpected manifest repo: %s", res.Manifest.Repo)
		}
		if res.Manifest.Branch != "feature-test" {
			t.Errorf("Unexpected manifest branch: %s", res.Manifest.Branch)
		}
	}

	// Verify extracted files
	helloPath := filepath.Join(destDir, "hello.txt")
	if data, err := os.ReadFile(helloPath); err != nil || string(data) != "hello world" {
		t.Errorf("hello.txt content mismatch or missing: %v", err)
	}

	// Verify .git was skipped
	gitPath := filepath.Join(destDir, ".git", "config")
	if _, err := os.Stat(gitPath); !os.IsNotExist(err) {
		t.Errorf(".git directory should have been skipped during extraction")
	}
}
