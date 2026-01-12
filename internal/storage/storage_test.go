package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielmerja/TamaLLM/internal/game"
)

func TestStorage_SaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tamallm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "test_save.json")
	storage := NewWithPath(savePath)

	// Create a state
	state := game.NewState("TestPet", "blob", 12345)
	state.Hunger = 50
	state.AddMemory("test", "This is a test")

	// Save
	err = storage.Save(state)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify file exists
	if !storage.Exists() {
		t.Error("Save file should exist")
	}

	// Load
	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify loaded state
	if loaded.Name != "TestPet" {
		t.Errorf("Expected name 'TestPet', got '%s'", loaded.Name)
	}
	if loaded.Hunger != 50 {
		t.Errorf("Expected hunger 50, got %d", loaded.Hunger)
	}
	if len(loaded.Memory) != 1 {
		t.Errorf("Expected 1 memory event, got %d", len(loaded.Memory))
	}
}

func TestStorage_LoadNoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tamallm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "nonexistent.json")
	storage := NewWithPath(savePath)

	_, err = storage.Load()
	if err != ErrNoSaveFile {
		t.Errorf("Expected ErrNoSaveFile, got %v", err)
	}
}

func TestStorage_LoadCorrupted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tamallm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "corrupted.json")

	// Write invalid JSON
	err = os.WriteFile(savePath, []byte("this is not json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	storage := NewWithPath(savePath)

	_, err = storage.Load()
	if err != ErrCorruptedSave {
		t.Errorf("Expected ErrCorruptedSave, got %v", err)
	}
}

func TestStorage_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tamallm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "to_delete.json")
	storage := NewWithPath(savePath)

	// Create a file
	state := game.NewState("Test", "blob", 0)
	err = storage.Save(state)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Delete
	err = storage.Delete()
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deleted
	if storage.Exists() {
		t.Error("Save file should not exist after delete")
	}
}

func TestStorage_DeleteNonexistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tamallm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "nonexistent.json")
	storage := NewWithPath(savePath)

	// Should not error when deleting nonexistent file
	err = storage.Delete()
	if err != nil {
		t.Errorf("Delete of nonexistent file should not error: %v", err)
	}
}

func TestStorage_GetSavePath(t *testing.T) {
	storage := NewWithPath("/some/path/save.json")

	if storage.GetSavePath() != "/some/path/save.json" {
		t.Errorf("Expected '/some/path/save.json', got '%s'", storage.GetSavePath())
	}
}
