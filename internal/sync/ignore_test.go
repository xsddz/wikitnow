package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnorer_ShouldIgnore(t *testing.T) {
	// Create a temporary directory structure for testing if needed
	tempDir := t.TempDir()

	// Create a fake .wikitnow/ignore to test compilation
	wikitnowDir := filepath.Join(tempDir, ".wikitnow")
	os.MkdirAll(wikitnowDir, 0755)
	ignorePath := filepath.Join(wikitnowDir, "ignore")

	content := []byte("secret.txt\n*.log\n")
	os.WriteFile(ignorePath, content, 0644)

	// Init Ignorer in the temp directory
	ignorer := NewIgnorer(tempDir)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{
			name:     "Hidden file",
			path:     "/Users/foo/.hidden",
			isDir:    false,
			expected: true, // Should fail on prefix '.'
		},
		{
			name:     "Custom ignore rule (secret.txt)",
			path:     filepath.Join(tempDir, "secret.txt"),
			isDir:    false,
			expected: true,
		},
		{
			name:     "Custom ignore rule glob (*.log)",
			path:     filepath.Join(tempDir, "app.log"),
			isDir:    false,
			expected: true,
		},
		{
			name:     "Valid normal file",
			path:     filepath.Join(tempDir, "main.go"),
			isDir:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignorer.ShouldIgnore(tt.path, tt.isDir)
			if got != tt.expected {
				t.Errorf("ShouldIgnore() got = %v, expected %v, path %v", got, tt.expected, tt.path)
			}
		})
	}
}
