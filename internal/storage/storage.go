// Package storage handles save/load operations for TamaLLM.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danielmerja/TamaLLM/internal/game"
)

const (
	SchemaVersion = 1
	SaveFileName  = "tamallm_save.json"
)

// SaveData wraps the game state with metadata.
type SaveData struct {
	SchemaVersion int         `json:"schema_version"`
	SavedAt       time.Time   `json:"saved_at"`
	State         *game.State `json:"state"`
}

// Storage manages persistence operations.
type Storage struct {
	savePath string
}

// New creates a new Storage instance using XDG-friendly paths.
func New() (*Storage, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Storage{
		savePath: filepath.Join(dataDir, SaveFileName),
	}, nil
}

// NewWithPath creates a Storage with a custom save path.
func NewWithPath(path string) *Storage {
	return &Storage{savePath: path}
}

// getDataDir returns the appropriate data directory following XDG spec.
func getDataDir() (string, error) {
	// Check XDG_DATA_HOME first
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "tamallm"), nil
	}

	// Fall back to ~/.local/share
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".local", "share", "tamallm"), nil
}

// Save persists the game state to disk.
func (s *Storage) Save(state *game.State) error {
	data := SaveData{
		SchemaVersion: SchemaVersion,
		SavedAt:       time.Now(),
		State:         state,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first, then rename for atomic operation
	tempPath := s.savePath + ".tmp"
	if err := os.WriteFile(tempPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	if err := os.Rename(tempPath, s.savePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to finalize save: %w", err)
	}

	return nil
}

// Load reads the game state from disk.
func (s *Storage) Load() (*game.State, error) {
	jsonData, err := os.ReadFile(s.savePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSaveFile
		}
		return nil, fmt.Errorf("failed to read save file: %w", err)
	}

	var data SaveData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, ErrCorruptedSave
	}

	// Version migration could happen here
	if data.SchemaVersion > SchemaVersion {
		return nil, ErrNewerVersion
	}

	if data.State == nil {
		return nil, ErrCorruptedSave
	}

	return data.State, nil
}

// Exists checks if a save file exists.
func (s *Storage) Exists() bool {
	_, err := os.Stat(s.savePath)
	return err == nil
}

// Delete removes the save file.
func (s *Storage) Delete() error {
	if err := os.Remove(s.savePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete save file: %w", err)
	}
	return nil
}

// GetSavePath returns the current save path.
func (s *Storage) GetSavePath() string {
	return s.savePath
}

// Error types
var (
	ErrNoSaveFile    = errors.New("no save file found")
	ErrCorruptedSave = errors.New("save file is corrupted")
	ErrNewerVersion  = errors.New("save file is from a newer version")
)
