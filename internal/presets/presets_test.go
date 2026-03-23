package presets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidHookType(t *testing.T) {
	validTypes := []string{"precreate", "postcreate", "prestart", "poststart", "presubmit", "postsubmit", "prereset", "postreset"}

	for _, hookType := range validTypes {
		if !IsValidHookType(hookType) {
			t.Errorf("Expected '%s' to be valid", hookType)
		}
	}

	invalidTypes := []string{"invalid", "", "pre_create", "POSTCREATE"}
	for _, hookType := range invalidTypes {
		if IsValidHookType(hookType) {
			t.Errorf("Expected '%s' to be invalid", hookType)
		}
	}
}

func TestGetHookNames(t *testing.T) {
	names := GetHookNames()

	expected := []string{"git-reset", "git-commit", "notify", "log"}
	if len(names) != len(expected) {
		t.Errorf("Expected %d hook names, got %d", len(expected), len(names))
	}

	for _, exp := range expected {
		found := false
		for _, name := range names {
			if name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected hook name '%s' not found", exp)
		}
	}
}

func TestInstallHook_InvalidHookType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = InstallHook(tmpDir, "git-commit", "", false)
	if err == nil {
		t.Error("Expected error for empty hook type")
	}

	_, err = InstallHook(tmpDir, "git-commit", "invalid", false)
	if err == nil {
		t.Error("Expected error for invalid hook type")
	}
}

func TestInstallHook_UnknownHook(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = InstallHook(tmpDir, "unknown", "prestart", false)
	if err == nil {
		t.Error("Expected error for unknown hook")
	}
}

func TestInstallHook_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath, err := InstallHook(tmpDir, "git-commit", "postcreate", false)
	if err != nil {
		t.Fatalf("InstallHook failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "postcreate", "git-commit.sh")
	if scriptPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, scriptPath)
	}

	// Verify file exists and is executable
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("Failed to stat hook file: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("Hook file should be executable")
	}
}

func TestInstallHook_Overwrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First install
	_, err = InstallHook(tmpDir, "git-commit", "postcreate", false)
	if err != nil {
		t.Fatalf("First InstallHook failed: %v", err)
	}

	// Second install without force should fail
	_, err = InstallHook(tmpDir, "git-commit", "postcreate", false)
	if err == nil {
		t.Error("Expected error when overwriting without force")
	}

	// Third install with force should succeed
	_, err = InstallHook(tmpDir, "git-commit", "postcreate", true)
	if err != nil {
		t.Fatalf("InstallHook with force failed: %v", err)
	}
}

func TestGenerateSkillFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "presets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	skillPath, err := GenerateSkillFile(tmpDir, "mytask")
	if err != nil {
		t.Fatalf("GenerateSkillFile failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "skills", "mytask", "SKILL.md")
	if skillPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, skillPath)
	}

	// Verify file exists
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("Skill file not found: %v", err)
	}

	// Second generate should fail (file exists)
	_, err = GenerateSkillFile(tmpDir, "mytask")
	if err == nil {
		t.Error("Expected error when skill file already exists")
	}
}

func TestPredefinedHooks_Content(t *testing.T) {
	for name, template := range PredefinedHooks {
		if template.Filename == "" {
			t.Errorf("Hook '%s' has empty filename", name)
		}
		if template.Content == "" {
			t.Errorf("Hook '%s' has empty content", name)
		}
		if template.Content[0:2] != "#!" {
			t.Errorf("Hook '%s' content should start with shebang", name)
		}
	}
}