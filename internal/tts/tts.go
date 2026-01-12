// Package tts provides Text-to-Speech integration for TamaLLM.
// It supports the Supertonic-2 model from Hugging Face for fast, on-device speech synthesis.
package tts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config holds TTS client configuration.
type Config struct {
	Enabled     bool          // Whether TTS is enabled
	VoiceStyle  string        // Voice style (M1-M5, F1-F5)
	Speed       float64       // Speech speed multiplier
	Timeout     time.Duration // Synthesis timeout
	CacheDir    string        // Directory for caching audio files
	AutoPlay    bool          // Whether to automatically play generated audio
}

// DefaultConfig returns the default TTS configuration.
func DefaultConfig() Config {
	cacheDir := os.TempDir()
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		cacheDir = filepath.Join(xdgCache, "tamallm", "tts")
	} else if home, err := os.UserHomeDir(); err == nil {
		cacheDir = filepath.Join(home, ".cache", "tamallm", "tts")
	}

	return Config{
		Enabled:    false, // Disabled by default, user needs to enable
		VoiceStyle: "F3",  // Default female voice
		Speed:      1.0,
		Timeout:    30 * time.Second,
		CacheDir:   cacheDir,
		AutoPlay:   true,
	}
}

// Client is the interface for TTS operations.
type Client interface {
	// Speak synthesizes speech from text and plays it.
	Speak(ctx context.Context, text string) error
	// SpeakAsync synthesizes speech from text asynchronously.
	SpeakAsync(text string)
	// IsAvailable checks if the TTS service is available.
	IsAvailable() bool
	// Stop stops any currently playing audio.
	Stop()
	// StatusInfo returns human-readable status information for debugging.
	StatusInfo() string
}

// SupertonicClient implements TTS using Supertonic-2 via Python.
// Since Supertonic requires Python/ONNX runtime, we use a subprocess approach.
type SupertonicClient struct {
	config     Config
	mu         sync.Mutex
	available  bool
	checkOnce  sync.Once
	currentCmd *exec.Cmd
	useUV      bool   // whether to use "uv run" to invoke python
	pythonCmd  string // the python command to use (python3 or python)
	lastError  string // last error message for debugging
}

// NewSupertonicClient creates a new Supertonic TTS client.
func NewSupertonicClient(config Config) *SupertonicClient {
	// Validate and sanitize voice style (must be M1-M5 or F1-F5)
	if !isValidVoiceStyle(config.VoiceStyle) {
		config.VoiceStyle = "F3" // Default to F3 if invalid
	}

	client := &SupertonicClient{
		config: config,
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		// Not fatal, just means caching won't work
	}

	return client
}

// isValidVoiceStyle checks if the voice style is valid (M1-M5 or F1-F5).
func isValidVoiceStyle(style string) bool {
	validStyles := map[string]bool{
		"M1": true, "M2": true, "M3": true, "M4": true, "M5": true,
		"F1": true, "F2": true, "F3": true, "F4": true, "F5": true,
	}
	return validStyles[style]
}

// checkSupertonic checks if supertonic is available and determines the Python command to use.
// It tries uv run python3 first (for uv-managed environments), then falls back to direct python3 or python.
// Returns (available, useUV, pythonCmd, errorMsg).
func checkSupertonic() (available bool, useUV bool, pythonCmd string, errorMsg string) {
	// First, try uv run python3 (handles uv-managed environments created with "uv venv" and "uv pip install")
	cmd := exec.Command("uv", "run", "python3", "-c", "import supertonic; print('ok')")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "ok" {
		return true, true, "python3", ""
	}
	uvError := ""
	if err != nil {
		uvError = err.Error()
	}

	// Fall back to direct python3
	cmd = exec.Command("python3", "-c", "import supertonic; print('ok')")
	output, err = cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "ok" {
		return true, false, "python3", ""
	}
	python3Error := ""
	if err != nil {
		python3Error = err.Error()
	}

	// Try python (some systems use python instead of python3)
	cmd = exec.Command("python", "-c", "import supertonic; print('ok')")
	output, err = cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "ok" {
		return true, false, "python", ""
	}
	pythonError := ""
	if err != nil {
		pythonError = err.Error()
	}

	// Build error message with details about what was tried
	// Format errors more clearly, only showing non-empty errors
	var errs []string
	if uvError != "" {
		errs = append(errs, "uv run python3: "+uvError)
	}
	if python3Error != "" {
		errs = append(errs, "python3: "+python3Error)
	}
	if pythonError != "" {
		errs = append(errs, "python: "+pythonError)
	}
	errDetails := "no errors captured"
	if len(errs) > 0 {
		errDetails = strings.Join(errs, "; ")
	}
	return false, false, "", fmt.Sprintf("supertonic not found (%s)", errDetails)
}

// IsAvailable checks if Supertonic is installed and available.
func (c *SupertonicClient) IsAvailable() bool {
	c.checkOnce.Do(func() {
		// Check if Python and supertonic module are available
		// This also determines whether to use uv or direct python/python3
		c.available, c.useUV, c.pythonCmd, c.lastError = checkSupertonic()
	})
	return c.available
}

// Speak synthesizes and plays speech from text.
func (c *SupertonicClient) Speak(ctx context.Context, text string) error {
	if !c.config.Enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop any currently playing audio
	c.stopLocked()

	// Clean text for speech (remove emoticons and special characters that don't speak well)
	cleanText := cleanTextForSpeech(text)
	if cleanText == "" {
		return nil
	}

	// Create a Python script that reads text from stdin (safer than embedding in script)
	script := fmt.Sprintf(`
import sys
try:
    from supertonic import TTS
    import soundfile as sf
    import tempfile
    import os

    # Check for audio playback capability
    try:
        import sounddevice as sd
        has_audio = True
    except:
        has_audio = False

    tts = TTS(auto_download=True)
    style = tts.get_voice_style('%s')
    
    # Read text from stdin for safety
    text = sys.stdin.read().strip()
    if not text:
        sys.exit(0)
    
    wav, duration = tts.synthesize(text, voice_style=style, total_steps=3)
    
    if has_audio:
        sd.play(wav.squeeze(), tts.sample_rate)
        sd.wait()
    else:
        # Save to temp file for external playback
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as f:
            sf.write(f.name, wav.squeeze(), tts.sample_rate)
            print(f.name)
    
    print('done')
except Exception as e:
    print(f'error: {e}', file=sys.stderr)
    sys.exit(1)
`, c.config.VoiceStyle)

	// Use uv run if available (for uv-managed environments), otherwise direct python/python3
	var cmd *exec.Cmd
	// pythonCmd is set by IsAvailable() which must be called before Speak()
	// The fallback to "python3" is defensive and shouldn't normally be reached
	pythonCmd := c.pythonCmd
	if pythonCmd == "" {
		pythonCmd = "python3"
	}
	if c.useUV {
		cmd = exec.CommandContext(ctx, "uv", "run", pythonCmd, "-c", script)
	} else {
		cmd = exec.CommandContext(ctx, pythonCmd, "-c", script)
	}
	c.currentCmd = cmd

	// Pass text via stdin to avoid injection issues
	cmd.Stdin = strings.NewReader(cleanText)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("TTS synthesis failed: %s", stderr.String())
	}

	return nil
}

// SpeakAsync synthesizes and plays speech asynchronously.
func (c *SupertonicClient) SpeakAsync(text string) {
	if !c.config.Enabled || !c.IsAvailable() {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
		defer cancel()
		_ = c.Speak(ctx, text)
	}()
}

// Stop stops any currently playing audio.
func (c *SupertonicClient) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}

func (c *SupertonicClient) stopLocked() {
	if c.currentCmd != nil && c.currentCmd.Process != nil {
		_ = c.currentCmd.Process.Kill()
		c.currentCmd = nil
	}
}

// StatusInfo returns human-readable status information for debugging.
func (c *SupertonicClient) StatusInfo() string {
	if !c.config.Enabled {
		return "TTS disabled"
	}
	// Trigger availability check if not done yet
	available := c.IsAvailable()
	if available {
		method := c.pythonCmd
		if c.useUV {
			method = "uv run " + c.pythonCmd
		}
		return fmt.Sprintf("Available (via %s, voice: %s)", method, c.config.VoiceStyle)
	}
	if c.lastError != "" {
		return fmt.Sprintf("Unavailable: %s", c.lastError)
	}
	return "Unavailable (supertonic module not found)"
}

// MockClient is a mock implementation for testing and no-TTS mode.
type MockClient struct {
	lastText string
}

// NewMockClient creates a new mock TTS client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// Speak does nothing in mock mode.
func (m *MockClient) Speak(ctx context.Context, text string) error {
	m.lastText = text
	return nil
}

// SpeakAsync does nothing in mock mode.
func (m *MockClient) SpeakAsync(text string) {
	m.lastText = text
}

// IsAvailable always returns true for mock.
func (m *MockClient) IsAvailable() bool {
	return true
}

// Stop does nothing in mock mode.
func (m *MockClient) Stop() {}

// StatusInfo returns status information for mock client.
func (m *MockClient) StatusInfo() string {
	return "TTS disabled (mock client)"
}

// LastText returns the last text that was "spoken" (for testing).
func (m *MockClient) LastText() string {
	return m.lastText
}

// cleanTextForSpeech removes characters that don't synthesize well.
func cleanTextForSpeech(text string) string {
	// Remove common emoticons and special characters
	replacements := []struct {
		old string
		new string
	}{
		{"*", ""},
		{"~", ""},
		{"_", ""},
		{"😊", ""},
		{"🎮", ""},
		{"🍎", ""},
		{"🍪", ""},
		{"😴", ""},
		{"💊", ""},
		{"👏", ""},
		{"😠", ""},
		{"🎉", ""},
		{"💤", ""},
		{"🤒", ""},
		{"Zzz", ""},
		{"zzz", ""},
	}

	result := text
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.old, r.new)
	}

	// Clean up extra spaces
	result = strings.Join(strings.Fields(result), " ")
	return strings.TrimSpace(result)
}

// escapeForPython escapes a string for use in a Python triple-quoted string.
func escapeForPython(s string) string {
	// Escape backslashes and triple quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'''", "\\'\\'\\'")
	return s
}
