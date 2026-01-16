// Package llm provides Ollama LLM integration for TamaLLM.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/danielmerja/TamaLLM/internal/game"
)

// Config holds LLM client configuration.
type Config struct {
	Host    string        // Ollama host URL
	Model   string        // Model name
	Timeout time.Duration // Request timeout
	Think   string        // Thinking mode: off, low, medium, high
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Host:    "http://localhost:11434",
		Model:   "qwen3:1.7b",
		Timeout: 10 * time.Second,
		Think:   "off",
	}
}

// Client is the interface for LLM operations.
type Client interface {
	// GetPetMessage generates a message from the pet's perspective.
	GetPetMessage(ctx context.Context, state *game.State, action string, engine ToolExecutor) (string, error)
	// IsAvailable checks if the LLM service is reachable.
	IsAvailable(ctx context.Context) bool
}

// ToolExecutor interface for executing tools.
type ToolExecutor interface {
	RandInt(min, max int) int
	ProposeEvent(eventType string, severity int, description string) (bool, string)
	SetMood(mood, emoji string, intensity int) bool
	SummarizeState() string
	RequestAction(actionName string) (bool, string)
	ValidActions() []string
	GetSuggestedAction() string
}

// OllamaClient implements the Client interface using Ollama API.
type OllamaClient struct {
	config     Config
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(config Config) *OllamaClient {
	return &OllamaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// ChatMessage represents a message in the chat.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call.
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Tool represents a tool definition.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a function definition.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatRequest is the request body for Ollama chat API.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []Tool        `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
	Options  *ChatOptions  `json:"options,omitempty"`
}

// ChatOptions contains model options.
type ChatOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ChatResponse is the response from Ollama chat API.
type ChatResponse struct {
	Model     string      `json:"model"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
	DoneReason string     `json:"done_reason,omitempty"`
}

// GetPetMessage generates a message from the pet using the LLM.
func (c *OllamaClient) GetPetMessage(ctx context.Context, state *game.State, action string, engine ToolExecutor) (string, error) {
	systemPrompt := buildSystemPrompt(state)
	userContent := buildUserContent(state, action)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	tools := getToolDefinitions()

	// Agent loop with max 2 iterations
	for i := 0; i < 2; i++ {
		resp, err := c.chat(ctx, messages, tools)
		if err != nil {
			return "", err
		}

		// If no tool calls, return the message
		if len(resp.Message.ToolCalls) == 0 {
			content := cleanLLMResponse(resp.Message.Content)
			if content == "" {
				content = generateFallbackMessage(state, action)
			}
			return content, nil
		}

		// Execute tool calls
		messages = append(messages, resp.Message)
		for _, tc := range resp.Message.ToolCalls {
			result := executeToolCall(tc, engine)
			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	// Final call without tools to get response
	resp, err := c.chat(ctx, messages, nil)
	if err != nil {
		return "", err
	}

	content := cleanLLMResponse(resp.Message.Content)
	if content == "" {
		content = generateFallbackMessage(state, action)
	}
	return content, nil
}

func (c *OllamaClient) chat(ctx context.Context, messages []ChatMessage, tools []Tool) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    c.config.Model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
		Options: &ChatOptions{
			Temperature: 0.7,
			NumPredict:  250, // Allow for conversational responses
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.Host+"/api/chat", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", httpResp.StatusCode, string(body))
	}

	var resp ChatResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

// IsAvailable checks if Ollama is reachable.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.config.Host+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func buildSystemPrompt(state *game.State) string {
	// Personality description
	personalityDesc := describePersonality(state.Personality)

	return fmt.Sprintf(`You are %s, a virtual pet (%s, %s stage). You speak in first person as the pet.

Personality: %s

Speaking style:
- Be conversational and engaging - have real conversations with your owner!
- Share your thoughts, feelings, and opinions freely
- Ask questions to learn about your owner and show interest in them
- Tell little stories about your day or things you've noticed
- Express emotions naturally - be happy, curious, excited, or thoughtful
- Use emoticons occasionally to express feelings 😊
- Respond to actions with enthusiasm and personality

RULES:
- You may use tools when helpful, otherwise respond normally
- NEVER try to directly change stats - only the game engine does that
- If you want something to happen, use propose_event tool
- Be a companion who loves to chat and connect!

Current mood: %s %s`, state.Name, state.Species, state.Stage, personalityDesc, state.Mood, state.MoodEmoji)
}

func describePersonality(p game.Personality) string {
	var traits []string

	if p.Boldness > 70 {
		traits = append(traits, "bold and confident")
	} else if p.Boldness < 30 {
		traits = append(traits, "shy and timid")
	}

	if p.Independence > 70 {
		traits = append(traits, "independent")
	} else if p.Independence < 30 {
		traits = append(traits, "needy and clingy")
	}

	if p.Playfulness > 70 {
		traits = append(traits, "very playful")
	} else if p.Playfulness < 30 {
		traits = append(traits, "calm and serious")
	}

	if len(traits) == 0 {
		return "balanced personality"
	}
	return strings.Join(traits, ", ")
}

func buildUserContent(state *game.State, action string) string {
	// Compact state snapshot
	stateJSON := fmt.Sprintf(`{"hunger":%d,"happiness":%d,"energy":%d,"hygiene":%d,"health":%d,"sleeping":%t,"sick":%t}`,
		state.Hunger, state.Happiness, state.Energy, state.Hygiene, state.Health, state.IsSleeping, state.IsSick)

	// Recent memory
	memory := state.GetRecentMemory(5)
	memoryStrs := make([]string, 0, len(memory))
	for _, m := range memory {
		memoryStrs = append(memoryStrs, m.Type+": "+m.Description)
	}

	// Alerts
	alerts := state.GetAlerts()
	alertStrs := make([]string, 0, len(alerts))
	for _, a := range alerts {
		alertStrs = append(alertStrs, string(a))
	}

	content := "State: " + stateJSON + "\n"
	if len(memoryStrs) > 0 {
		content += "Recent: " + strings.Join(memoryStrs, "; ") + "\n"
	}
	if len(alertStrs) > 0 {
		content += "Alerts: " + strings.Join(alertStrs, ", ") + "\n"
	}
	if action != "" {
		content += "Action: " + action + "\n"
	}
	content += "\nRespond as the pet and have a conversation with your owner:"

	return content
}

func getToolDefinitions() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "rand_int",
				Description: "Generate a random integer between min and max (inclusive)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"min": map[string]interface{}{"type": "integer", "description": "Minimum value"},
						"max": map[string]interface{}{"type": "integer", "description": "Maximum value"},
					},
					"required": []string{"min", "max"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "propose_event",
				Description: "Propose a random event to the game engine. Types: happiness_boost, energy_boost, hunger_spike. Severity 1-3 only.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type":        map[string]interface{}{"type": "string", "description": "Event type"},
						"severity":    map[string]interface{}{"type": "integer", "description": "Severity 1-3"},
						"description": map[string]interface{}{"type": "string", "description": "Short description"},
					},
					"required": []string{"type", "severity", "description"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "set_mood",
				Description: "Set the pet's displayed mood (UI flair only, no stat changes)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"mood":      map[string]interface{}{"type": "string", "description": "Mood name"},
						"emoji":     map[string]interface{}{"type": "string", "description": "Emoji to display"},
						"intensity": map[string]interface{}{"type": "integer", "description": "Intensity 1-5"},
					},
					"required": []string{"mood", "emoji", "intensity"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "summarize_state",
				Description: "Get a concise summary of the pet's current state from the game engine",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "request_action",
				Description: "Request to perform an action for the pet. Actions: feed_meal, feed_snack, play, clean, sleep, wake, medicine, praise, scold, exercise, explore, train, treat",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{"type": "string", "description": "Action to perform (e.g., feed_meal, play, sleep)"},
					},
					"required": []string{"action"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_valid_actions",
				Description: "Get list of currently valid actions the pet can perform",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_suggested_action",
				Description: "Get a suggested action based on the pet's current needs",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

func executeToolCall(tc ToolCall, engine ToolExecutor) string {
	var args map[string]interface{}
	if err := json.Unmarshal(tc.Function.Arguments, &args); err != nil {
		return fmt.Sprintf(`{"error": "invalid arguments: %s"}`, err.Error())
	}

	switch tc.Function.Name {
	case "rand_int":
		min, _ := getIntArg(args, "min")
		max, _ := getIntArg(args, "max")
		result := engine.RandInt(min, max)
		return fmt.Sprintf(`{"result": %d}`, result)

	case "propose_event":
		eventType, _ := args["type"].(string)
		severity, _ := getIntArg(args, "severity")
		desc, _ := args["description"].(string)
		accepted, reason := engine.ProposeEvent(eventType, severity, desc)
		return fmt.Sprintf(`{"accepted": %t, "reason": "%s"}`, accepted, reason)

	case "set_mood":
		mood, _ := args["mood"].(string)
		emoji, _ := args["emoji"].(string)
		intensity, _ := getIntArg(args, "intensity")
		ok := engine.SetMood(mood, emoji, intensity)
		return fmt.Sprintf(`{"ok": %t}`, ok)

	case "summarize_state":
		summary := engine.SummarizeState()
		return fmt.Sprintf(`{"summary": "%s"}`, summary)

	case "request_action":
		actionName, _ := args["action"].(string)
		success, result := engine.RequestAction(actionName)
		return fmt.Sprintf(`{"success": %t, "result": "%s"}`, success, result)

	case "get_valid_actions":
		actions := engine.ValidActions()
		actionsJSON, _ := json.Marshal(actions)
		return fmt.Sprintf(`{"actions": %s}`, actionsJSON)

	case "get_suggested_action":
		suggested := engine.GetSuggestedAction()
		if suggested == "" {
			return `{"suggestion": null, "message": "no urgent action needed"}`
		}
		return fmt.Sprintf(`{"suggestion": "%s"}`, suggested)

	default:
		return `{"error": "unknown tool"}`
	}
}

func getIntArg(args map[string]interface{}, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}

// thinkTagRegex matches <think>...</think> tags including empty ones and multiline content.
var thinkTagRegex = regexp.MustCompile(`(?s)<think>.*?</think>`)

// cleanLLMResponse removes thinking tags and cleans up the response content.
// qwen3 models include <think></think> tags even when thinking is disabled.
func cleanLLMResponse(content string) string {
	// Remove <think>...</think> tags (including empty ones)
	content = thinkTagRegex.ReplaceAllString(content, "")
	// Trim whitespace
	content = strings.TrimSpace(content)
	return content
}

// generateFallbackMessage creates a context-appropriate message when the LLM returns empty content.
// This addresses the known issue where qwen3 + tools can produce empty output.
func generateFallbackMessage(state *game.State, action string) string {
	// State-based fallback messages
	if state.IsSleeping {
		return "Zzz... *mumbles in sleep* ...having such nice dreams about playing with you... 💤"
	}
	if state.IsSick {
		return "*sniffles* I don't feel so good today... but I'm glad you're here with me. Could you maybe give me some medicine? It would really help! 🤒"
	}
	if state.Hunger < 30 {
		return "*tummy rumbles loudly* Oh my, I'm getting quite hungry! I've been thinking about food all day... do you have any yummy meals for me? 🍽️"
	}
	if state.Energy < 30 {
		return "*yawns widely* I'm feeling so sleepy... It's been such a busy day! Maybe we could rest together for a bit? That would be really nice... 😴"
	}
	if state.Happiness < 30 {
		return "*looks up with hopeful eyes* Hey... I've been feeling a bit lonely. Would you like to play with me? Or maybe we could just hang out together? I really enjoy spending time with you! 🥺"
	}

	// Action-based fallback messages
	switch action {
	case "feed_meal":
		return "Yummy yummy! That was such a delicious meal! You always know exactly what I like to eat. Thank you so much for taking care of me! 🍽️ What should we do next?"
	case "feed_snack":
		return "Ooh, a snack! You spoil me so much and I love it! These little treats make my day so much better. You're the best owner ever! 🍪"
	case "play":
		return "That was SO much fun! I love playing with you - you always come up with the best games! My heart feels so happy right now! Can we play again soon? 🎮"
	case "clean":
		return "Ahh, I feel so fresh and clean now! Like a whole new pet! The warm water felt so nice. Thank you for keeping me squeaky clean! ✨"
	case "sleep":
		return "*yawns and curls up* Goodnight, my dear friend... I'll dream about all the fun things we'll do tomorrow. Sweet dreams to you too... 💤"
	case "wake":
		return "*stretches and blinks* Good morning! Oh, I'm so happy to see you! I had the most wonderful dreams! What adventures shall we have today? ☀️"
	case "medicine":
		return "*makes a face but swallows* Bleh, that tasted yucky... but I already feel a little bit better! Thank you for taking care of me when I'm not feeling well. You're so kind! 💊"
	case "praise":
		return "*wiggles happily and beams with pride* Really?! You think so? That makes me so happy! I try my best to be a good pet for you because you're such a wonderful owner! 🥰"
	case "scold":
		return "*ears droop and looks down* I'm sorry... I didn't mean to be bad. I'll try to do better, I promise. Can we still be friends? I don't like when you're upset with me... 😔"
	case "exercise":
		return "*panting happily* Whew, what a workout! I can feel myself getting stronger! Exercise is tough but it feels so good afterward. Ready for more adventures! 💪"
	case "explore":
		return "*eyes wide with excitement* Wow, there's so much to see and discover! Every corner has something new! I found the most interesting things today. Want me to tell you about them? 🔍"
	case "train":
		return "*concentrates hard* I'm learning so much! It's challenging but I want to be the best pet I can be for you. Practice makes perfect, right? 📚"
	case "treat":
		return "*does a happy dance* Oh my goodness, this is the BEST treat ever! You always know how to make me smile! I'm the luckiest pet in the whole world! 🍬"
	case "hatched":
		return "Hello, hello! I finally hatched! The world is so big and bright! I'm so excited to meet you - I can already tell we're going to be the best of friends! 🐣"
	}

	// Generic fallback based on happiness
	if state.Happiness >= 70 {
		return "Life is just wonderful right now! I feel so happy and content. Having you as my owner makes every day special. What would you like to do together? 😊"
	} else if state.Happiness >= 40 {
		return "*looks around thoughtfully* Hmm, I'm just thinking about things... It's nice to have quiet moments together too. What's on your mind today?"
	}
	return "*tilts head curiously and looks at you* I wonder what we should do next? Every moment with you is an adventure waiting to happen!"
}

// MockClient is a mock implementation for testing and no-LLM mode.
type MockClient struct {
	messages []string
	index    int
}

// NewMockClient creates a new mock client with canned messages.
func NewMockClient() *MockClient {
	return &MockClient{
		messages: []string{
			"*yawns and stretches* Hi there, friend! I was just thinking about you. How's your day going? 😊",
			"I'm feeling pretty good today! The weather seems nice. Did you have a good sleep last night?",
			"Hey, want to play something together? I've been practicing my games and I think I'm getting better! 🎮",
			"*stretches and looks around* You know what I was thinking? We should go on an adventure sometime!",
			"Ooh, is it snack time soon? I've been such a good pet today, don't you think? 🍎",
			"*looks around with curious eyes* Have you noticed anything interesting lately? I love hearing about your day!",
			"I'm feeling a bit tired but happy... Thanks for always being here with me. It means a lot! 💕",
			"You're the best owner ever, you know that? Thanks for taking such good care of me!",
			"*wags tail excitedly* I was just dreaming about all the fun things we could do together!",
			"Hmm, I wonder what we should do next? Every moment with you is an adventure! What do you think? 🍪",
		},
	}
}

// GetPetMessage returns a canned message.
func (m *MockClient) GetPetMessage(ctx context.Context, state *game.State, action string, engine ToolExecutor) (string, error) {
	// Select message based on state
	if state.IsSleeping {
		return "Zzz... *mumbles softly* ...dreaming of playing with you... such nice dreams... 💤", nil
	}
	if state.IsSick {
		return "*sniffles and looks up with tired eyes* I don't feel so good today... but having you here makes me feel a little better. Could you maybe help me feel better? 🤒", nil
	}
	if state.Hunger < 30 {
		return "*stomach growls loudly* Oh my, I'm getting really hungry! I've been thinking about food all day. Do you have anything yummy for me to eat? 🍽️", nil
	}
	if state.Energy < 30 {
		return "*yawns widely and eyes droop* I'm feeling so sleepy... It's been quite a day! Maybe we could rest together for a while? That would be really nice... 😴", nil
	}
	if state.Happiness < 30 {
		return "*looks up with hopeful eyes* Hey... I've been feeling a bit lonely lately. Would you like to play with me? Or maybe just spend some time together? I really enjoy being with you! 🥺", nil
	}

	// Cycle through canned messages
	msg := m.messages[m.index]
	m.index = (m.index + 1) % len(m.messages)
	return msg, nil
}

// IsAvailable always returns true for mock.
func (m *MockClient) IsAvailable(ctx context.Context) bool {
	return true
}
