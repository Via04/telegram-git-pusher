package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest defines repo metadata inside a zip file (.tg-git.yaml or .tg-git.json)
type Manifest struct {
	Repo        string `yaml:"repo" json:"repo"`
	Branch      string `yaml:"branch" json:"branch"`
	Message     string `yaml:"message" json:"message"`
	SSHKey      string `yaml:"ssh_key" json:"ssh_key"`
	Token       string `yaml:"token" json:"token"`
	AuthorName  string `yaml:"author_name" json:"author_name"`
	AuthorEmail string `yaml:"author_email" json:"author_email"`
}

// ExtractResult holds extraction output and any manifest found.
type ExtractResult struct {
	Manifest *Manifest
	Files    []string
}

// ExtractZip extracts a zip file to destDir, replacing existing files.
// It skips any top-level or nested `.git` directory inside the archive.
// Returns list of extracted files and any parsed manifest (.tg-git.yaml / .tg-git.json).
func ExtractZip(zipPath, destDir string) (*ExtractResult, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	destDirClean := filepath.Clean(destDir)
	var manifest *Manifest
	var extractedFiles []string

	// Detect if all files inside zip are wrapped in a single root directory (e.g. repo-main/)
	rootDirPrefix := detectSingleRootDir(r.File)

	for _, f := range r.File {
		// Normalize file path
		filePath := f.Name

		// Remove single root directory wrapper if detected
		if rootDirPrefix != "" {
			if !strings.HasPrefix(filePath, rootDirPrefix) {
				continue
			}
			filePath = strings.TrimPrefix(filePath, rootDirPrefix)
		}

		if filePath == "" {
			continue
		}

		// Prevent Zip Slip vulnerability
		targetPath := filepath.Join(destDirClean, filePath)
		if !strings.HasPrefix(targetPath, destDirClean+string(os.PathSeparator)) && targetPath != destDirClean {
			return nil, fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		// Skip .git directory from zip
		relPath := filepath.ToSlash(filePath)
		if strings.HasPrefix(relPath, ".git/") || relPath == ".git" || strings.Contains(relPath, "/.git/") {
			continue
		}

		// Check for Manifest file (.tg-git.yaml or .tg-git.json or tg-git.yaml)
		baseName := filepath.Base(filePath)
		if (baseName == ".tg-git.yaml" || baseName == ".tg-git.json" || baseName == "tg-git.yaml") && manifest == nil {
			rc, err := f.Open()
			if err == nil {
				data, err := io.ReadAll(rc)
				rc.Close()
				if err == nil {
					var m Manifest
					if err := yaml.Unmarshal(data, &m); err == nil {
						manifest = &m
					}
				}
			}
			// Skip copying the bot manifest itself to target git repo
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}

		// Ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent dir for %s: %w", targetPath, err)
		}

		// Create/Overwrite destination file
		out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return nil, fmt.Errorf("failed to open zipped file %s: %w", f.Name, err)
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to copy content to %s: %w", targetPath, err)
		}

		extractedFiles = append(extractedFiles, filePath)
	}

	return &ExtractResult{
		Manifest: manifest,
		Files:    extractedFiles,
	}, nil
}

// detectSingleRootDir checks if all files in the archive are under a single top-level directory.
func detectSingleRootDir(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}

	var firstDir string
	for _, f := range files {
		parts := strings.Split(filepath.ToSlash(f.Name), "/")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		if firstDir == "" {
			firstDir = parts[0]
		} else if firstDir != parts[0] {
			return "" // Multiple root directories/files found
		}
	}

	// Check if firstDir is actually a directory (not just a single top-level file)
	if firstDir != "" {
		for _, f := range files {
			if f.Name == firstDir {
				// It's a file, not a dir wrapper
				return ""
			}
		}
		return firstDir + "/"
	}

	return ""
}
