package tts

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("TTS should be disabled by default")
	}
	if config.VoiceStyle == "" {
		t.Error("VoiceStyle should have a default value")
	}
	if config.Speed != 1.0 {
		t.Errorf("Expected default speed of 1.0, got %v", config.Speed)
	}
	if config.Timeout == 0 {
		t.Error("Timeout should have a default value")
	}
	if config.CacheDir == "" {
		t.Error("CacheDir should have a default value")
	}
}

func TestMockClient_Speak(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	err := client.Speak(ctx, "Hello world")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if client.LastText() != "Hello world" {
		t.Errorf("Expected 'Hello world', got '%s'", client.LastText())
	}
}

func TestMockClient_SpeakAsync(t *testing.T) {
	client := NewMockClient()

	client.SpeakAsync("Test async")
	time.Sleep(10 * time.Millisecond) // Give async a moment

	if client.LastText() != "Test async" {
		t.Errorf("Expected 'Test async', got '%s'", client.LastText())
	}
}

func TestMockClient_IsAvailable(t *testing.T) {
	client := NewMockClient()

	if !client.IsAvailable() {
		t.Error("MockClient should always be available")
	}
}

func TestMockClient_Stop(t *testing.T) {
	client := NewMockClient()
	// Stop should not panic
	client.Stop()
}

func TestCleanTextForSpeech(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"*yawns* Hello", "yawns Hello"},
		{"😊 Happy!", "Happy!"},
		{"Zzz... sleeping", "... sleeping"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"", ""},
		{"🎮🍎🍪", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanTextForSpeech(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTextForSpeech(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeForPython(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"It's fine", "It's fine"},
		{"with'''quotes", "with\\'\\'\\'quotes"},
		{"back\\slash", "back\\\\slash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeForPython(tt.input)
			if result != tt.expected {
				t.Errorf("escapeForPython(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSupertonicClient_DisabledDoesNothing(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	client := NewSupertonicClient(config)
	ctx := context.Background()

	// Should return nil even though Supertonic might not be installed
	err := client.Speak(ctx, "Hello")
	if err != nil {
		t.Errorf("Disabled client should not error: %v", err)
	}
}

func TestSupertonicClient_Stop(t *testing.T) {
	config := DefaultConfig()
	client := NewSupertonicClient(config)

	// Stop should not panic even with no active process
	client.Stop()
}

func TestIsValidVoiceStyle(t *testing.T) {
	validStyles := []string{"M1", "M2", "M3", "M4", "M5", "F1", "F2", "F3", "F4", "F5"}
	invalidStyles := []string{"X1", "M6", "F0", "", "invalid", "m1", "f1"}

	for _, style := range validStyles {
		if !isValidVoiceStyle(style) {
			t.Errorf("Expected %q to be valid", style)
		}
	}

	for _, style := range invalidStyles {
		if isValidVoiceStyle(style) {
			t.Errorf("Expected %q to be invalid", style)
		}
	}
}

func TestSupertonicClient_InvalidVoiceStyleDefaultsToF3(t *testing.T) {
	config := DefaultConfig()
	config.VoiceStyle = "invalid"

	client := NewSupertonicClient(config)

	// Client should have defaulted to F3
	if client.config.VoiceStyle != "F3" {
		t.Errorf("Expected voice style to default to F3, got %q", client.config.VoiceStyle)
	}
}

func TestCheckSupertonic(t *testing.T) {
	// This test just verifies the function returns consistent values
	// The actual behavior depends on whether uv and/or supertonic are installed
	available, useUV, pythonCmd, errMsg := checkSupertonic()

	// If available, useUV should be either true or false, errMsg should be empty
	// If not available, useUV should be false, errMsg should contain info
	if !available && useUV {
		t.Error("useUV should be false when supertonic is not available")
	}
	if available && errMsg != "" {
		t.Error("errMsg should be empty when supertonic is available")
	}
	if !available && errMsg == "" {
		t.Error("errMsg should contain info when supertonic is not available")
	}
	if available && pythonCmd == "" {
		t.Error("pythonCmd should not be empty when supertonic is available")
	}
	if !available && pythonCmd != "" {
		t.Error("pythonCmd should be empty when supertonic is not available")
	}
}

func TestSupertonicClient_UseUVFlag(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	client := NewSupertonicClient(config)

	// Call IsAvailable to set useUV flag
	_ = client.IsAvailable()

	// Verify the useUV flag matches what checkSupertonic returns
	_, expectedUseUV, _, _ := checkSupertonic()
	if client.useUV != expectedUseUV {
		t.Errorf("Expected useUV to be %v, got %v", expectedUseUV, client.useUV)
	}
}

func TestSupertonicClient_StatusInfo(t *testing.T) {
	// Test disabled client
	config := DefaultConfig()
	config.Enabled = false
	client := NewSupertonicClient(config)
	status := client.StatusInfo()
	if status != "TTS disabled" {
		t.Errorf("Expected 'TTS disabled', got '%s'", status)
	}

	// Test enabled client (availability depends on environment)
	config.Enabled = true
	client = NewSupertonicClient(config)
	status = client.StatusInfo()
	// Status should contain useful info either way
	if status == "" {
		t.Error("StatusInfo should not be empty")
	}
}

func TestMockClient_StatusInfo(t *testing.T) {
	client := NewMockClient()
	status := client.StatusInfo()
	if status != "TTS disabled (mock client)" {
		t.Errorf("Expected 'TTS disabled (mock client)', got '%s'", status)
	}
}

func TestIsPlaceholderText(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"...", true},
		{"..", true},
		{".", true},
		{"(none)", true},
		{"", true},
		{"Hello world", false},
		{"...hello", false},
		{"Some actual message", false},
		{"*yawns* Good morning!", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isPlaceholderText(tt.input)
			if result != tt.expected {
				t.Errorf("isPlaceholderText(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
