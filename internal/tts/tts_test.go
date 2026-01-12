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
