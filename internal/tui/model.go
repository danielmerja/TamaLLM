// Package tui implements the terminal user interface for TamaLLM.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/danielmerja/TamaLLM/internal/game"
	"github.com/danielmerja/TamaLLM/internal/llm"
	"github.com/danielmerja/TamaLLM/internal/storage"
	"github.com/danielmerja/TamaLLM/internal/tts"
)

// Screen represents the current screen being displayed.
type Screen int

const (
	ScreenMain Screen = iota
	ScreenMenu
	ScreenMinigame
	ScreenSettings
	ScreenHelp
	ScreenNewPet
	ScreenGraveyard
	ScreenRecovery
)

// LLM Auto mode configuration
const (
	LLMAutoActionCooldown = 15 // Ticks between LLM auto actions
)

// Model is the main Bubble Tea model.
type Model struct {
	// Core components
	engine  *game.Engine
	llm     llm.Client
	tts     tts.Client
	storage *storage.Storage

	// UI state
	screen        Screen
	menuIndex     int
	width         int
	height        int
	debugMode     bool
	lastLLMReq    string
	lastLLMResp   string

	// New pet form
	nameInput textinput.Model

	// Minigame state
	minigameType   string
	minigameTarget int
	minigameGuess  int
	minigameResult string

	// Progress bars
	hungerBar    progress.Model
	happinessBar progress.Model
	energyBar    progress.Model
	hygieneBar   progress.Model
	healthBar    progress.Model

	// Messages
	statusMessage string
	petMessage    string

	// Timing
	tickInterval  time.Duration
	lastSave      time.Time
	saveDebounce  time.Duration

	// LLM config
	llmEnabled     bool
	llmPending     bool
	llmAutoMode    bool
	lastLLMAction  int64

	// TTS config
	ttsEnabled     bool

	// Error state
	errorMessage string
}

// Config holds configuration for the TUI.
type Config struct {
	TickInterval time.Duration
	LLMEnabled   bool
	LLMAutoMode  bool
	LLMConfig    llm.Config
	TTSEnabled   bool
	TTSConfig    tts.Config
	StartNew     bool
}

// DefaultConfig returns default TUI configuration.
func DefaultConfig() Config {
	return Config{
		TickInterval: time.Second,
		LLMEnabled:   true,
		LLMAutoMode:  false,
		LLMConfig:    llm.DefaultConfig(),
		TTSEnabled:   false,
		TTSConfig:    tts.DefaultConfig(),
		StartNew:     false,
	}
}

// New creates a new TUI model.
func New(config Config, store *storage.Storage) Model {
	// Initialize progress bars
	hungerBar := progress.New(progress.WithDefaultGradient())
	happinessBar := progress.New(progress.WithGradient("#FFD700", "#32CD32"))
	energyBar := progress.New(progress.WithGradient("#00BFFF", "#1E90FF"))
	hygieneBar := progress.New(progress.WithGradient("#87CEEB", "#4169E1"))
	healthBar := progress.New(progress.WithGradient("#FF6347", "#FF0000"))

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter pet name"
	ti.CharLimit = 20
	ti.Width = 20

	// Initialize LLM client
	var llmClient llm.Client
	if config.LLMEnabled {
		llmClient = llm.NewOllamaClient(config.LLMConfig)
	} else {
		llmClient = llm.NewMockClient()
	}

	// Initialize TTS client
	var ttsClient tts.Client
	if config.TTSEnabled {
		ttsConfig := config.TTSConfig
		ttsConfig.Enabled = true
		ttsClient = tts.NewSupertonicClient(ttsConfig)
	} else {
		ttsClient = tts.NewMockClient()
	}

	m := Model{
		storage:       store,
		llm:           llmClient,
		tts:           ttsClient,
		screen:        ScreenMain,
		hungerBar:     hungerBar,
		happinessBar:  happinessBar,
		energyBar:     energyBar,
		hygieneBar:    hygieneBar,
		healthBar:     healthBar,
		nameInput:     ti,
		tickInterval:  config.TickInterval,
		saveDebounce:  5 * time.Second,
		llmEnabled:    config.LLMEnabled,
		llmAutoMode:   config.LLMAutoMode,
		ttsEnabled:    config.TTSEnabled,
		petMessage:    "...",
	}

	return m
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadOrCreatePet(),
		tea.SetWindowTitle("TamaLLM - Virtual Pet"),
	)
}

func (m Model) loadOrCreatePet() tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return newPetMsg{}
		}

		state, err := m.storage.Load()
		if err != nil {
			if err == storage.ErrNoSaveFile {
				return newPetMsg{}
			}
			if err == storage.ErrCorruptedSave {
				return recoveryMsg{err: err}
			}
			return newPetMsg{}
		}

		return loadedPetMsg{state: state}
	}
}

// Messages
type (
	tickMsg        time.Time
	loadedPetMsg   struct{ state *game.State }
	newPetMsg      struct{}
	recoveryMsg    struct{ err error }
	savedMsg       struct{ err error }
	llmResponseMsg struct {
		message string
		err     error
	}
)

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update progress bar widths
		barWidth := min(40, m.width-20)
		m.hungerBar.Width = barWidth
		m.happinessBar.Width = barWidth
		m.energyBar.Width = barWidth
		m.hygieneBar.Width = barWidth
		m.healthBar.Width = barWidth

	case tickMsg:
		if m.engine != nil && m.engine.State.Alive {
			m.engine.Tick()

			// Autosave periodically
			if time.Since(m.lastSave) > m.saveDebounce {
				cmds = append(cmds, m.save())
			}

			// Request LLM message periodically
			if !m.llmPending && m.engine.State.Age%10 == 0 {
				m.llmPending = true
				m.lastLLMReq = ""
				cmds = append(cmds, m.requestLLMMessage(""))
			}

			// Auto mode: LLM-driven actions
			if m.llmAutoMode && !m.llmPending && m.engine.State.Age-m.lastLLMAction >= LLMAutoActionCooldown {
				suggested := m.engine.GetSuggestedAction()
				if suggested != "" {
					m.lastLLMAction = m.engine.State.Age
					success, result := m.engine.RequestAction(suggested)
					if success {
						m.statusMessage = fmt.Sprintf("[AUTO] %s: %s", suggested, result)
						m.llmPending = true
						m.lastLLMReq = suggested
						cmds = append(cmds, m.requestLLMMessage(suggested))
					}
				}
			}
		}
		cmds = append(cmds, m.scheduleTick())

	case loadedPetMsg:
		m.engine = game.NewEngine(msg.state)
		m.petMessage = "Welcome back! I missed you!"
		cmds = append(cmds, m.scheduleTick())

	case newPetMsg:
		m.screen = ScreenNewPet
		m.nameInput.Focus()
		return m, textinput.Blink

	case recoveryMsg:
		m.screen = ScreenRecovery
		m.errorMessage = msg.err.Error()

	case savedMsg:
		if msg.err != nil {
			m.statusMessage = "Save failed: " + msg.err.Error()
		} else {
			m.lastSave = time.Now()
		}

	case llmResponseMsg:
		m.llmPending = false
		if msg.err != nil {
			m.lastLLMResp = "Error: " + msg.err.Error()
		} else {
			m.petMessage = msg.message
			m.lastLLMResp = msg.message
			// Speak the message via TTS if enabled
			if m.ttsEnabled && m.tts != nil {
				m.tts.SpeakAsync(msg.message)
			}
		}

	case progress.FrameMsg:
		// Update all progress bars
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.hungerBar.Update(msg)
		m.hungerBar = model.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch {
	case key.Matches(msg, keys.Quit):
		if m.engine != nil {
			_ = m.storage.Save(m.engine.State)
		}
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		if m.screen == ScreenHelp {
			m.screen = ScreenMain
		} else {
			m.screen = ScreenHelp
		}
		return m, nil

	case key.Matches(msg, keys.Debug):
		m.debugMode = !m.debugMode
		return m, nil

	case key.Matches(msg, keys.Auto):
		if m.llmEnabled && m.screen == ScreenMain {
			m.llmAutoMode = !m.llmAutoMode
			if m.llmAutoMode {
				m.statusMessage = "Auto mode enabled - LLM will make decisions"
			} else {
				m.statusMessage = "Auto mode disabled"
			}
		}
		return m, nil
	}

	// Screen-specific keys
	switch m.screen {
	case ScreenNewPet:
		return m.updateNewPetScreen(msg)
	case ScreenMain:
		return m.updateMainScreen(msg)
	case ScreenMenu:
		return m.updateMenuScreen(msg)
	case ScreenMinigame:
		return m.updateMinigameScreen(msg)
	case ScreenHelp:
		return m.updateHelpScreen(msg)
	case ScreenGraveyard:
		return m.updateGraveyardScreen(msg)
	case ScreenRecovery:
		return m.updateRecoveryScreen(msg)
	}

	return m, nil
}

func (m Model) updateNewPetScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			name = "Tama"
		}
		state := game.NewState(name, "blob", 0)
		m.engine = game.NewEngine(state)
		m.screen = ScreenMain
		m.petMessage = "Hello! I'm " + name + "! Nice to meet you!"
		m.llmPending = true
		m.lastLLMReq = "hatched"
		return m, tea.Batch(m.scheduleTick(), m.save(), m.requestLLMMessage("hatched"))

	case tea.KeyEsc:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m Model) updateMainScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Select), key.Matches(msg, keys.Menu):
		if m.engine != nil && m.engine.State.Alive {
			m.screen = ScreenMenu
			m.menuIndex = 0
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateMenuScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	menuItems := m.getMenuItems()

	switch {
	case key.Matches(msg, keys.Up):
		m.menuIndex--
		if m.menuIndex < 0 {
			m.menuIndex = len(menuItems) - 1
		}

	case key.Matches(msg, keys.Down):
		m.menuIndex++
		if m.menuIndex >= len(menuItems) {
			m.menuIndex = 0
		}

	case key.Matches(msg, keys.Select):
		return m.executeMenuAction(menuItems[m.menuIndex])

	case key.Matches(msg, keys.Back):
		m.screen = ScreenMain
	}

	return m, nil
}

func (m Model) updateMinigameScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		m.minigameGuess++
		if m.minigameGuess > 10 {
			m.minigameGuess = 1
		}

	case key.Matches(msg, keys.Down):
		m.minigameGuess--
		if m.minigameGuess < 1 {
			m.minigameGuess = 10
		}

	case key.Matches(msg, keys.Select):
		// Check guess
		if m.minigameGuess == m.minigameTarget {
			m.minigameResult = "You got it! 🎉"
			m.engine.State.Happiness += 20
			if m.engine.State.Happiness > 100 {
				m.engine.State.Happiness = 100
			}
		} else {
			m.minigameResult = fmt.Sprintf("Nope! It was %d", m.minigameTarget)
			m.engine.State.Happiness += 5
			if m.engine.State.Happiness > 100 {
				m.engine.State.Happiness = 100
			}
		}
		m.engine.State.Energy -= 15
		if m.engine.State.Energy < 0 {
			m.engine.State.Energy = 0
		}
		m.engine.State.AddMemory("minigame", "Played number guessing game")

	case key.Matches(msg, keys.Back):
		m.screen = ScreenMain
		m.llmPending = true
		m.lastLLMReq = "finished playing"
		return m, m.requestLLMMessage("finished playing")
	}

	return m, nil
}

func (m Model) updateHelpScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.Back) || msg.Type == tea.KeyEsc {
		m.screen = ScreenMain
	}
	return m, nil
}

func (m Model) updateGraveyardScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Select):
		// Start new game
		if m.storage != nil {
			_ = m.storage.Delete()
		}
		m.screen = ScreenNewPet
		m.nameInput.Reset()
		m.nameInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) updateRecoveryScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Select):
		// Delete corrupted save and start fresh
		if m.storage != nil {
			_ = m.storage.Delete()
		}
		m.screen = ScreenNewPet
		m.nameInput.Reset()
		m.nameInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) getMenuItems() []menuItem {
	items := []menuItem{
		{name: "Feed Meal", action: game.ActionFeedMeal, icon: "🍽️"},
		{name: "Feed Snack", action: game.ActionFeedSnack, icon: "🍪"},
		{name: "Give Treat", action: game.ActionTreat, icon: "🍬"},
		{name: "Play", action: game.ActionPlay, icon: "🎮"},
		{name: "Exercise", action: game.ActionExercise, icon: "🏃"},
		{name: "Explore", action: game.ActionExplore, icon: "🔍"},
		{name: "Train", action: game.ActionTrain, icon: "📚"},
		{name: "Clean", action: game.ActionClean, icon: "🛁"},
	}

	if m.engine.State.IsSleeping {
		items = append(items, menuItem{name: "Wake Up", action: game.ActionWake, icon: "☀️"})
	} else {
		items = append(items, menuItem{name: "Sleep", action: game.ActionSleep, icon: "😴"})
	}

	if m.engine.State.IsSick {
		items = append(items, menuItem{name: "Medicine", action: game.ActionMedicine, icon: "💊"})
	}

	items = append(items,
		menuItem{name: "Praise", action: game.ActionPraise, icon: "👏"},
		menuItem{name: "Scold", action: game.ActionScold, icon: "😠"},
		menuItem{name: "Back", action: "", icon: "◀️"},
	)

	return items
}

type menuItem struct {
	name   string
	action game.Action
	icon   string
}

func (m Model) executeMenuAction(item menuItem) (tea.Model, tea.Cmd) {
	if item.action == "" {
		m.screen = ScreenMain
		return m, nil
	}

	if item.action == game.ActionPlay {
		// Start minigame
		m.minigameType = "guess"
		m.minigameTarget = m.engine.RandInt(1, 10)
		m.minigameGuess = 5
		m.minigameResult = ""
		m.screen = ScreenMinigame
		return m, nil
	}

	// Execute action
	result := m.engine.PerformAction(item.action)
	m.statusMessage = result
	m.screen = ScreenMain

	// Check if pet died
	if !m.engine.State.Alive {
		m.screen = ScreenGraveyard
		return m, nil
	}

	m.llmPending = true
	m.lastLLMReq = string(item.action)
	return m, tea.Batch(m.save(), m.requestLLMMessage(string(item.action)))
}

func (m Model) scheduleTick() tea.Cmd {
	return tea.Tick(m.tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) save() tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil || m.engine == nil {
			return savedMsg{err: nil}
		}
		err := m.storage.Save(m.engine.State)
		return savedMsg{err: err}
	}
}

func (m Model) requestLLMMessage(action string) tea.Cmd {
	if m.llm == nil || m.engine == nil {
		return nil
	}

	// Note: m.llmPending and m.lastLLMReq must be set by caller before calling
	// this function, since this is a value receiver.

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		msg, err := m.llm.GetPetMessage(ctx, m.engine.State, action, m.engine)
		return llmResponseMsg{message: msg, err: err}
	}
}

// View renders the UI.
func (m Model) View() string {
	switch m.screen {
	case ScreenNewPet:
		return m.viewNewPet()
	case ScreenMain:
		return m.viewMain()
	case ScreenMenu:
		return m.viewMenu()
	case ScreenMinigame:
		return m.viewMinigame()
	case ScreenHelp:
		return m.viewHelp()
	case ScreenGraveyard:
		return m.viewGraveyard()
	case ScreenRecovery:
		return m.viewRecovery()
	default:
		return "Unknown screen"
	}
}

// Keybindings
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Select key.Binding
	Back   key.Binding
	Menu   key.Binding
	Help   key.Binding
	Debug  key.Binding
	Auto   key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Menu: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "menu"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Debug: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "debug"),
	),
	Auto: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "toggle auto mode"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
