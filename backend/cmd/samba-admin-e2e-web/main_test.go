package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanPathCannotEscapeFrontendRoot(t *testing.T) {
	root := t.TempDir()
	for _, requestPath := range []string{"../../etc/passwd", "assets/index.js", "index.html"} {
		candidate := filepath.Join(root, filepath.Clean(strings.TrimPrefix(requestPath, "/")))
		relative, err := filepath.Rel(root, candidate)
		escapes := err != nil || strings.HasPrefix(relative, "..")
		if strings.Contains(requestPath, "..") != escapes {
			t.Fatalf("classificação inesperada para %q: escapes=%v", requestPath, escapes)
		}
	}
}
