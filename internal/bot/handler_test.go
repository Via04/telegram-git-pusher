package bot

import (
	"testing"
)

func TestParseCaption(t *testing.T) {
	defaultRepo := "git@github.com:default/repo.git"
	defaultBranch := "main"

	// Case 1: Plain commit message when default repo is set
	r, b, m := parseCaption("Fix layout bug and update authentication", defaultRepo, defaultBranch)
	if r != defaultRepo || b != defaultBranch || m != "Fix layout bug and update authentication" {
		t.Errorf("Case 1 failed: got repo=%s branch=%s msg=%s", r, b, m)
	}

	// Case 2: Explicit prefix -m "message"
	r, b, m = parseCaption("-m \"Refactor database package\"", defaultRepo, defaultBranch)
	if m != "Refactor database package" {
		t.Errorf("Case 2 failed: got msg=%s", m)
	}

	// Case 3: Explicit prefix msg: message
	r, b, m = parseCaption("msg: Implemented new API endpoint", defaultRepo, defaultBranch)
	if m != "Implemented new API endpoint" {
		t.Errorf("Case 3 failed: got msg=%s", m)
	}

	// Case 4: Key-value caption
	r, b, m = parseCaption("repo=git@github.com:custom/repo.git branch=dev msg=\"Custom commit\"", defaultRepo, defaultBranch)
	if r != "git@github.com:custom/repo.git" || b != "dev" || m != "Custom commit" {
		t.Errorf("Case 4 failed: got repo=%s branch=%s msg=%s", r, b, m)
	}

	// Case 5: Full positional URL caption
	r, b, m = parseCaption("git@github.com:custom/repo.git feature/login \"Positional commit message\"", "", "master")
	if r != "git@github.com:custom/repo.git" || b != "feature/login" || m != "Positional commit message" {
		t.Errorf("Case 5 failed: got repo=%s branch=%s msg=%s", r, b, m)
	}

	// Case 6: Empty caption uses default message
	r, b, m = parseCaption("", defaultRepo, defaultBranch)
	if m != "Update code via Telegram Bot" {
		t.Errorf("Case 6 failed: got msg=%s", m)
	}
}
