package llm

import (
	"context"
	"testing"

	"github.com/danielmerja/TamaLLM/internal/game"
)

// MockToolExecutor implements ToolExecutor for testing.
type MockToolExecutor struct {
	randIntResult       int
	proposeAccepted     bool
	proposeReason       string
	setMoodOK           bool
	summarizeResult     string
	requestActionResult string
	requestActionOK     bool
	validActions        []string
	suggestedAction     string
}

func (m *MockToolExecutor) RandInt(min, max int) int {
	return m.randIntResult
}

func (m *MockToolExecutor) ProposeEvent(eventType string, severity int, description string) (bool, string) {
	return m.proposeAccepted, m.proposeReason
}

func (m *MockToolExecutor) SetMood(mood, emoji string, intensity int) bool {
	return m.setMoodOK
}

func (m *MockToolExecutor) SummarizeState() string {
	return m.summarizeResult
}

func (m *MockToolExecutor) RequestAction(actionName string) (bool, string) {
	return m.requestActionOK, m.requestActionResult
}

func (m *MockToolExecutor) ValidActions() []string {
	return m.validActions
}

func (m *MockToolExecutor) GetSuggestedAction() string {
	return m.suggestedAction
}

func TestMockClient_GetPetMessage(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	executor := &MockToolExecutor{}
	ctx := context.Background()

	// Normal state should return a canned message
	msg, err := client.GetPetMessage(ctx, state, "", executor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg == "" {
		t.Error("Expected non-empty message")
	}
}

func TestMockClient_SleepingMessage(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	state.IsSleeping = true
	executor := &MockToolExecutor{}
	ctx := context.Background()

	msg, err := client.GetPetMessage(ctx, state, "", executor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg != "Zzz... *mumbles in sleep* ...having such nice dreams about playing with you... 💤" {
		t.Errorf("Expected sleeping message, got: %s", msg)
	}
}

func TestMockClient_SickMessage(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	state.IsSick = true
	executor := &MockToolExecutor{}
	ctx := context.Background()

	msg, err := client.GetPetMessage(ctx, state, "", executor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg != "*sniffles* I don't feel so good today... but I'm glad you're here with me. Could you maybe give me some medicine? It would really help! 🤒" {
		t.Errorf("Expected sick message, got: %s", msg)
	}
}

func TestMockClient_HungryMessage(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	state.Hunger = 20
	executor := &MockToolExecutor{}
	ctx := context.Background()

	msg, err := client.GetPetMessage(ctx, state, "", executor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg != "*tummy rumbles loudly* Oh my, I'm getting quite hungry! I've been thinking about food all day... do you have any yummy meals for me? 🍽️" {
		t.Errorf("Expected hungry message, got: %s", msg)
	}
}

func TestMockClient_TiredMessage(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	state.Energy = 20
	executor := &MockToolExecutor{}
	ctx := context.Background()

	msg, err := client.GetPetMessage(ctx, state, "", executor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg != "*yawns widely* I'm feeling so sleepy... It's been such a busy day! Maybe we could rest together for a bit? That would be really nice... 😴" {
		t.Errorf("Expected tired message, got: %s", msg)
	}
}

func TestMockClient_SadMessage(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	state.Happiness = 20
	executor := &MockToolExecutor{}
	ctx := context.Background()

	msg, err := client.GetPetMessage(ctx, state, "", executor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if msg != "*looks up with hopeful eyes* Hey... I've been feeling a bit lonely. Would you like to play with me? Or maybe we could just hang out together? I really enjoy spending time with you! 🥺" {
		t.Errorf("Expected sad message, got: %s", msg)
	}
}

func TestMockClient_IsAvailable(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	if !client.IsAvailable(ctx) {
		t.Error("MockClient should always be available")
	}
}

func TestMockClient_MessageCycling(t *testing.T) {
	client := NewMockClient()
	state := game.NewState("Test", "blob", 0)
	executor := &MockToolExecutor{}
	ctx := context.Background()

	// Get multiple messages to test cycling
	messages := make([]string, 3)
	for i := 0; i < 3; i++ {
		msg, _ := client.GetPetMessage(ctx, state, "", executor)
		messages[i] = msg
	}

	// Should cycle through different messages
	if messages[0] == messages[1] && messages[1] == messages[2] {
		t.Error("Messages should cycle through different options")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "http://localhost:11434" {
		t.Errorf("Expected default host, got: %s", config.Host)
	}
	if config.Model != "qwen3:1.7b" {
		t.Errorf("Expected default model, got: %s", config.Model)
	}
	if config.Timeout.Seconds() != 10 {
		t.Errorf("Expected 10s timeout, got: %v", config.Timeout)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	state := game.NewState("Tama", "blob", 0)
	prompt := buildSystemPrompt(state)

	if prompt == "" {
		t.Error("System prompt should not be empty")
	}
	if len(prompt) > 2000 {
		t.Error("System prompt should be reasonably sized")
	}
}

func TestBuildUserContent(t *testing.T) {
	state := game.NewState("Tama", "blob", 0)
	state.AddMemory("test", "Test event")

	content := buildUserContent(state, "feed_meal")

	if content == "" {
		t.Error("User content should not be empty")
	}
	// Should contain state info
	if len(content) < 50 {
		t.Error("User content seems too short")
	}
}

func TestDescribePersonality(t *testing.T) {
	tests := []struct {
		name     string
		p        game.Personality
		contains string
	}{
		{"Bold", game.Personality{Boldness: 80, Independence: 50, Playfulness: 50}, "bold"},
		{"Shy", game.Personality{Boldness: 20, Independence: 50, Playfulness: 50}, "shy"},
		{"Independent", game.Personality{Boldness: 50, Independence: 80, Playfulness: 50}, "independent"},
		{"Needy", game.Personality{Boldness: 50, Independence: 20, Playfulness: 50}, "needy"},
		{"Playful", game.Personality{Boldness: 50, Independence: 50, Playfulness: 80}, "playful"},
		{"Serious", game.Personality{Boldness: 50, Independence: 50, Playfulness: 20}, "serious"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := describePersonality(tt.p)
			if desc == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}

func TestGetToolDefinitions(t *testing.T) {
	tools := getToolDefinitions()

	if len(tools) != 7 {
		t.Errorf("Expected 7 tools, got %d", len(tools))
	}

	expectedNames := []string{"rand_int", "propose_event", "set_mood", "summarize_state", "request_action", "get_valid_actions", "get_suggested_action"}
	for i, tool := range tools {
		if tool.Function.Name != expectedNames[i] {
			t.Errorf("Tool %d: expected name '%s', got '%s'", i, expectedNames[i], tool.Function.Name)
		}
		if tool.Type != "function" {
			t.Errorf("Tool %d: expected type 'function', got '%s'", i, tool.Type)
		}
	}
}

func TestExecuteToolCall_RandInt(t *testing.T) {
	executor := &MockToolExecutor{randIntResult: 42}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "rand_int",
			Arguments: []byte(`{"min": 1, "max": 100}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"result": 42}` {
		t.Errorf("Expected result 42, got: %s", result)
	}
}

func TestExecuteToolCall_ProposeEvent(t *testing.T) {
	executor := &MockToolExecutor{proposeAccepted: true, proposeReason: "accepted"}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "propose_event",
			Arguments: []byte(`{"type": "happiness_boost", "severity": 2, "description": "test"}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"accepted": true, "reason": "accepted"}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_SetMood(t *testing.T) {
	executor := &MockToolExecutor{setMoodOK: true}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "set_mood",
			Arguments: []byte(`{"mood": "happy", "emoji": "😊", "intensity": 3}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"ok": true}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_SummarizeState(t *testing.T) {
	executor := &MockToolExecutor{summarizeResult: "Test summary"}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "summarize_state",
			Arguments: []byte(`{}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"summary": "Test summary"}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_UnknownTool(t *testing.T) {
	executor := &MockToolExecutor{}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "unknown_tool",
			Arguments: []byte(`{}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"error": "unknown tool"}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_InvalidArguments(t *testing.T) {
	executor := &MockToolExecutor{}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "rand_int",
			Arguments: []byte(`invalid json`),
		},
	}

	result := executeToolCall(tc, executor)
	if result == "" {
		t.Error("Expected error result for invalid arguments")
	}
}

func TestExecuteToolCall_RequestAction(t *testing.T) {
	executor := &MockToolExecutor{requestActionOK: true, requestActionResult: "action executed"}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "request_action",
			Arguments: []byte(`{"action": "feed_meal"}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"success": true, "result": "action executed"}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_GetValidActions(t *testing.T) {
	executor := &MockToolExecutor{validActions: []string{"feed_meal", "play", "sleep"}}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "get_valid_actions",
			Arguments: []byte(`{}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"actions": ["feed_meal","play","sleep"]}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_GetSuggestedAction(t *testing.T) {
	executor := &MockToolExecutor{suggestedAction: "feed_meal"}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "get_suggested_action",
			Arguments: []byte(`{}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"suggestion": "feed_meal"}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestExecuteToolCall_GetSuggestedAction_None(t *testing.T) {
	executor := &MockToolExecutor{suggestedAction: ""}
	tc := ToolCall{
		Function: FunctionCall{
			Name:      "get_suggested_action",
			Arguments: []byte(`{}`),
		},
	}

	result := executeToolCall(tc, executor)
	if result != `{"suggestion": null, "message": "no urgent action needed"}` {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestCleanLLMResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "Hello world", "Hello world"},
		{"empty think tags", "<think></think>Hello world", "Hello world"},
		{"think tags with content", "<think>reasoning here</think>Hello world", "Hello world"},
		{"multiline think tags", "<think>\nsome\nreasoning\n</think>Hello world", "Hello world"},
		{"only think tags", "<think>thinking</think>", ""},
		{"empty think with newlines", "<think>\n\n</think>Response", "Response"},
		{"no think tags", "Just a response", "Just a response"},
		{"whitespace only after cleaning", "<think>test</think>   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanLLMResponse(tt.input)
			if result != tt.expected {
				t.Errorf("cleanLLMResponse(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateFallbackMessage(t *testing.T) {
	// Test sleeping state
	state := game.NewState("Test", "blob", 0)
	state.IsSleeping = true
	msg := generateFallbackMessage(state, "")
	if msg == "" || msg == "..." {
		t.Error("Expected non-empty fallback for sleeping state")
	}

	// Test sick state
	state = game.NewState("Test", "blob", 0)
	state.IsSick = true
	msg = generateFallbackMessage(state, "")
	if msg == "" || msg == "..." {
		t.Error("Expected non-empty fallback for sick state")
	}

	// Test hungry state
	state = game.NewState("Test", "blob", 0)
	state.Hunger = 20
	msg = generateFallbackMessage(state, "")
	if msg == "" || msg == "..." {
		t.Error("Expected non-empty fallback for hungry state")
	}

	// Test action-based fallback
	state = game.NewState("Test", "blob", 0)
	msg = generateFallbackMessage(state, "feed_meal")
	if msg == "" || msg == "..." {
		t.Error("Expected non-empty fallback for feed_meal action")
	}

	// Test generic fallback
	state = game.NewState("Test", "blob", 0)
	msg = generateFallbackMessage(state, "unknown_action")
	if msg == "" || msg == "..." {
		t.Error("Expected non-empty generic fallback")
	}
}
