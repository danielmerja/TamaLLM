package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/danielmerja/TamaLLM/internal/game"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF69B4")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)

	alertStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	happyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#32CD32"))

	sadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedMenuStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#FF69B4")).
				Bold(true)

	debugStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))
)

func (m Model) viewNewPet() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🥚 Welcome to TamaLLM! 🥚"))
	b.WriteString("\n\n")
	b.WriteString("A new egg has appeared!\n")
	b.WriteString("What will you name your pet?\n\n")
	b.WriteString(m.nameInput.View())
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("Press Enter to confirm, Esc to quit"))

	return boxStyle.Render(b.String())
}

func (m Model) viewMain() string {
	if m.engine == nil {
		return "Loading..."
	}

	var b strings.Builder
	s := m.engine.State

	// Title
	title := fmt.Sprintf("🐾 %s 🐾", s.Name)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Stage and age
	stageEmoji := getStageEmoji(s.Stage)
	age := formatAge(s.Age)
	b.WriteString(fmt.Sprintf("%s %s | Age: %s\n", stageEmoji, s.Stage, age))

	// ASCII pet
	b.WriteString("\n")
	b.WriteString(getASCIIPet(s.Stage, s.IsSleeping, s.IsSick, s.Mood))
	b.WriteString("\n\n")

	// Mood
	b.WriteString(fmt.Sprintf("Mood: %s %s\n\n", s.MoodEmoji, s.Mood))

	// Stats
	b.WriteString(m.renderStats())
	b.WriteString("\n")

	// Alerts
	alerts := s.GetAlerts()
	if len(alerts) > 0 {
		alertStr := "⚠️ "
		for i, a := range alerts {
			if i > 0 {
				alertStr += " | "
			}
			alertStr += string(a)
		}
		b.WriteString(alertStyle.Render(alertStr))
		b.WriteString("\n\n")
	}

	// Pet message
	msgStyle := happyStyle
	if s.Happiness < 50 {
		msgStyle = sadStyle
	}
	b.WriteString(msgStyle.Render("\"" + m.petMessage + "\""))
	b.WriteString("\n\n")

	// Status message
	if m.statusMessage != "" {
		b.WriteString(subtitleStyle.Render(m.statusMessage))
		b.WriteString("\n")
	}

	// Controls hint
	autoStatus := ""
	if m.llmAutoMode {
		autoStatus = " | a: Auto [ON]"
	} else if m.llmEnabled {
		autoStatus = " | a: Auto [OFF]"
	}
	b.WriteString(subtitleStyle.Render("Enter: Menu | ?: Help | d: Debug" + autoStatus + " | q: Quit"))

	// Debug panel
	if m.debugMode {
		b.WriteString("\n")
		b.WriteString(m.renderDebug())
	}

	return boxStyle.Render(b.String())
}

func (m Model) renderStats() string {
	if m.engine == nil {
		return ""
	}
	s := m.engine.State

	var b strings.Builder

	// Stat bars
	stats := []struct {
		name  string
		value int
		bar   string
		emoji string
	}{
		{"Hunger", s.Hunger, m.hungerBar.ViewAs(float64(s.Hunger) / 100), "🍽️"},
		{"Happy", s.Happiness, m.happinessBar.ViewAs(float64(s.Happiness) / 100), "😊"},
		{"Energy", s.Energy, m.energyBar.ViewAs(float64(s.Energy) / 100), "⚡"},
		{"Hygiene", s.Hygiene, m.hygieneBar.ViewAs(float64(s.Hygiene) / 100), "🛁"},
		{"Health", s.Health, m.healthBar.ViewAs(float64(s.Health) / 100), "❤️"},
	}

	for _, stat := range stats {
		b.WriteString(fmt.Sprintf("%s %-7s %s %3d%%\n", stat.emoji, stat.name+":", stat.bar, stat.value))
	}

	// Additional info
	b.WriteString(fmt.Sprintf("\n📏 Weight: %d | 📚 Discipline: %d%%", s.Weight, s.Discipline))

	if s.IsSick {
		b.WriteString(" | 🤒 SICK")
	}
	if s.IsSleeping {
		b.WriteString(" | 💤 Sleeping")
	}

	return b.String()
}

func (m Model) renderDebug() string {
	if m.engine == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("=== DEBUG ===\n")
	b.WriteString(fmt.Sprintf("LLM Enabled: %t | Pending: %t | Auto: %t\n", m.llmEnabled, m.llmPending, m.llmAutoMode))
	b.WriteString(fmt.Sprintf("Last Request: %s\n", m.lastLLMReq))
	b.WriteString(fmt.Sprintf("Last Response: %s\n", m.lastLLMResp))
	b.WriteString(fmt.Sprintf("Age: %d | Stage: %s\n", m.engine.State.Age, m.engine.State.Stage))
	b.WriteString(fmt.Sprintf("Care Score: %.1f | Sickness Events: %d\n",
		m.engine.State.AverageCareScore, m.engine.State.TotalSicknessEvents))
	if m.llmAutoMode {
		suggested := m.engine.GetSuggestedAction()
		if suggested != "" {
			b.WriteString(fmt.Sprintf("Suggested Action: %s\n", suggested))
		}
	}

	return debugStyle.Render(b.String())
}

func (m Model) viewMenu() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎮 Actions"))
	b.WriteString("\n\n")

	items := m.getMenuItems()
	for i, item := range items {
		cursor := "  "
		style := menuItemStyle
		if i == m.menuIndex {
			cursor = "▶ "
			style = selectedMenuStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%s %s", cursor, item.icon, item.name)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("↑↓: Navigate | Enter: Select | Esc: Back"))

	return boxStyle.Render(b.String())
}

func (m Model) viewMinigame() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🎯 Guess the Number!"))
	b.WriteString("\n\n")
	b.WriteString("I'm thinking of a number between 1 and 10...\n\n")

	b.WriteString(fmt.Sprintf("Your guess: [ %d ]\n\n", m.minigameGuess))

	if m.minigameResult != "" {
		if strings.Contains(m.minigameResult, "🎉") {
			b.WriteString(happyStyle.Render(m.minigameResult))
		} else {
			b.WriteString(sadStyle.Render(m.minigameResult))
		}
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle.Render("Press Esc to return"))
	} else {
		b.WriteString(subtitleStyle.Render("↑↓: Change guess | Enter: Submit | Esc: Give up"))
	}

	return boxStyle.Render(b.String())
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📖 Help"))
	b.WriteString("\n\n")

	helpItems := []struct {
		key  string
		desc string
	}{
		{"↑/k, ↓/j", "Navigate menus"},
		{"Enter/Space", "Select / Open menu"},
		{"Esc/Backspace", "Go back"},
		{"m", "Open action menu"},
		{"a", "Toggle LLM auto mode"},
		{"?", "Toggle help"},
		{"d", "Toggle debug mode"},
		{"q", "Quit (autosaves)"},
	}

	for _, item := range helpItems {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-15s", item.key)),
			helpDescStyle.Render(item.desc)))
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("🎮 Gameplay Tips"))
	b.WriteString("\n\n")
	tips := []string{
		"• Keep all stats above 30 for a healthy pet",
		"• Feed meals for hunger, snacks for happiness",
		"• Playing uses energy but boosts happiness",
		"• Exercise improves health and discipline",
		"• Explore for random discoveries",
		"• Train to increase discipline",
		"• Clean regularly to prevent sickness",
		"• Let your pet sleep when tired",
		"• Balance praise and scolding for discipline",
		"• Your pet evolves based on care quality!",
		"• Enable auto mode (a) for LLM-driven care",
	}
	for _, tip := range tips {
		b.WriteString(tip + "\n")
	}

	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Press Esc or ? to close"))

	return boxStyle.Render(b.String())
}

func (m Model) viewGraveyard() string {
	if m.engine == nil {
		return "..."
	}

	var b strings.Builder
	s := m.engine.State

	b.WriteString(titleStyle.Render("💀 Rest in Peace 💀"))
	b.WriteString("\n\n")

	// Tombstone ASCII art
	tombstone := `
    _______
   /       \
  |  R.I.P  |
  |         |
  | %s |
  |         |
  |_________|
    |     |
    |     |
`
	name := s.Name
	if len(name) > 7 {
		name = name[:7]
	}
	name = fmt.Sprintf("%-7s", name)
	b.WriteString(fmt.Sprintf(tombstone, name))

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Name: %s\n", s.Name))
	b.WriteString(fmt.Sprintf("Species: %s\n", s.Species))
	b.WriteString(fmt.Sprintf("Final Stage: %s", s.Stage))
	if s.AdultType != "" {
		b.WriteString(fmt.Sprintf(" (%s)", s.AdultType))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Lived for: %s\n", formatAge(s.Age)))
	b.WriteString(fmt.Sprintf("Care Score: %.1f%%\n", s.AverageCareScore))

	b.WriteString("\n")
	b.WriteString("Last words: ")
	if len(s.Memory) > 0 {
		b.WriteString(s.Memory[len(s.Memory)-1].Description)
	}
	b.WriteString("\n\n")

	b.WriteString(subtitleStyle.Render("Press Enter to start a new game"))

	return boxStyle.Render(b.String())
}

func (m Model) viewRecovery() string {
	var b strings.Builder

	b.WriteString(alertStyle.Render("⚠️ Save File Corrupted"))
	b.WriteString("\n\n")
	b.WriteString("Your save file appears to be corrupted.\n")
	b.WriteString(fmt.Sprintf("Error: %s\n\n", m.errorMessage))
	b.WriteString("Press Enter to start a new game.\n")
	b.WriteString("(The corrupted save will be deleted)")

	return boxStyle.Render(b.String())
}

// ASCII Art helpers

func getStageEmoji(stage game.Stage) string {
	switch stage {
	case game.StageEgg:
		return "🥚"
	case game.StageBaby:
		return "🐣"
	case game.StageChild:
		return "🐥"
	case game.StageTeen:
		return "🐤"
	case game.StageAdult:
		return "🐔"
	default:
		return "❓"
	}
}

func getASCIIPet(stage game.Stage, sleeping, sick bool, mood string) string {
	if sleeping {
		return `
    ╭─────╮
    │ ─ ─ │  Zzz
    │  ▽  │
    ╰─────╯
`
	}

	if sick {
		return `
    ╭─────╮
    │ x x │  *cough*
    │  ~  │
    ╰─────╯
`
	}

	switch stage {
	case game.StageEgg:
		return `
    ╭───╮
    │ ? │
    │   │
    ╰───╯
`
	case game.StageBaby:
		return `
    ╭───╮
    │^_^│
    ╰───╯
`
	case game.StageChild:
		return `
    ╭─────╮
    │ • • │
    │  ◡  │
    ╰─────╯
`
	case game.StageTeen:
		return `
   ╭──────╮
   │ ◉ ◉  │
   │  ◡   │
   │      │
   ╰──────╯
`
	case game.StageAdult:
		if mood == "happy" {
			return `
  ╭────────╮
  │ ★   ★  │
  │   ◡◡   │
  │        │
  ╰────────╯
     /\  /\
`
		}
		return `
  ╭────────╮
  │ ◉   ◉  │
  │   ◡    │
  │        │
  ╰────────╯
     /\  /\
`
	default:
		return `
    ╭───╮
    │? ?│
    ╰───╯
`
	}
}

func formatAge(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
