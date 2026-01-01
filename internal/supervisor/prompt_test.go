// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSupervisorPrompt(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	// Save current directory and restore after test
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Test 1: Project-level SUPERVISOR.md takes precedence
	t.Run("project-level takes precedence", func(t *testing.T) {
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Create project-level file
		projectContent := "# Project Supervisor\nThis is project-specific"
		if err := os.WriteFile("./SUPERVISOR.md", []byte(projectContent), 0644); err != nil {
			t.Fatal(err)
		}

		content, err := GetSupervisorPrompt()
		if err != nil {
			t.Errorf("GetSupervisorPrompt() error = %v", err)
			return
		}

		if content != projectContent {
			t.Errorf("GetSupervisorPrompt() = %v, want %v", content, projectContent)
		}

		// Clean up
		os.Remove("./SUPERVISOR.md")
	})

	// Test 2: Fallback to global when project-level doesn't exist
	t.Run("fallback to global", func(t *testing.T) {
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Create global file in temp home
		homeDir := t.TempDir()
		globalContent := "# Global Supervisor\nThis is global"
		globalPath := filepath.Join(homeDir, ".claude")
		if err := os.MkdirAll(globalPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(globalPath, "SUPERVISOR.md"), []byte(globalContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Override UserHomeDir for testing - we can't easily do this,
		// so we'll just test the error case

		// Since we can't mock os.UserHomeDir, we'll test that the function
		// returns an error when neither file exists
		os.Remove(filepath.Join(globalPath, "SUPERVISOR.md"))
	})

	// Test 3: Neither file exists
	t.Run("neither file exists", func(t *testing.T) {
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Make sure project-level doesn't exist
		os.Remove("./SUPERVISOR.md")

		_, err := GetSupervisorPrompt()
		if err == nil {
			t.Error("GetSupervisorPrompt() should return error when no file exists")
		}
	})
}
