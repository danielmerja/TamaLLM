# TamaLLM 🐾

A Tamagotchi-like terminal game with a local LLM "brain" powered by Ollama.

![TamaLLM Screenshot](https://user-images.githubusercontent.com/placeholder/tamallm-screenshot.png)

## Features

- **Real-time pet simulation** - Stats decay over time, evolution through life stages, and care-dependent outcomes
- **Beautiful Terminal UI** - Built with Charm's Bubble Tea framework with colorful stat bars and ASCII art
- **LLM-Powered Personality** - Your pet speaks with a unique personality using a local Ollama model
- **Tool Calling** - The LLM can propose events and set moods through a safe, bounded tool system
- **Persistent Save** - Your pet persists across sessions with automatic saving
- **Deterministic Gameplay** - Game rules are deterministic; the LLM only adds flavor

## Quick Start

### Prerequisites

1. **Go 1.21+** - [Install Go](https://go.dev/doc/install)
2. **Ollama** - [Install Ollama](https://ollama.ai)

### Setup

```bash
# Clone the repository
git clone https://github.com/danielmerja/TamaLLM.git
cd TamaLLM

# Pull the recommended model
ollama pull llama3.2:1b

# Build and run
go run ./cmd/tamallm
```

### Run Without LLM

If you don't have Ollama installed or want to play without AI features:

```bash
go run ./cmd/tamallm --no-llm
```

## Controls

| Key | Action |
|-----|--------|
| `Enter` / `Space` | Select / Open menu |
| `↑/k`, `↓/j` | Navigate menus |
| `Esc` / `Backspace` | Go back |
| `m` | Open action menu |
| `?` | Toggle help |
| `d` | Toggle debug mode |
| `q` | Quit (auto-saves) |

## Gameplay

### Life Stages

Your pet evolves through 5 stages based on age and care quality:

1. **Egg** 🥚 - Hatches after ~30 seconds
2. **Baby** 🐣 - Grows to child after ~2 minutes
3. **Child** 🐥 - Becomes teenager after ~5 minutes
4. **Teen** 🐤 - Evolves to adult after ~10 minutes
5. **Adult** 🐔 - Final form depends on care quality!

### Adult Types

The type of adult your pet becomes depends on how well you cared for them:

- **Healthy Adult** - Good stats, few sickness events, balanced discipline
- **Chubby Adult** - Overfed with too many snacks
- **Naughty Adult** - Low discipline from lack of training
- **Negligence Adult** - Poor overall care quality

### Stats

Keep these stats healthy (above 30) to keep your pet happy:

- **Hunger** 🍽️ - Feed meals and snacks
- **Happiness** 😊 - Play games and give praise
- **Energy** ⚡ - Let your pet sleep when tired
- **Hygiene** 🛁 - Clean regularly to prevent sickness
- **Health** ❤️ - Give medicine when sick

### Actions

| Action | Effect |
|--------|--------|
| Feed Meal | +30 Hunger, +5 Happiness, +1 Weight |
| Feed Snack | +10 Hunger, +15 Happiness, chance +1 Weight |
| Play | +25 Happiness, -20 Energy, -10 Hunger |
| Clean | +40 Hygiene, +5 Happiness |
| Sleep | Restores Energy over time |
| Medicine | Cures sickness, +20 Health |
| Praise | +10 Happiness, slight -2 Discipline |
| Scold | +10 Discipline, -15 Happiness |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `llama3.2:1b` | Model to use |
| `OLLAMA_TIMEOUT` | `10s` | Request timeout |
| `OLLAMA_THINK` | `off` | Thinking mode (off/low/medium/high) |

### CLI Flags

```bash
go run ./cmd/tamallm [flags]

Flags:
  --new           Start a new game (ignore existing save)
  --model NAME    Override Ollama model
  --host URL      Override Ollama host URL
  --tick-ms N     Simulation tick interval (default: 1000)
  --no-llm        Run without LLM (use canned messages)
```

### Recommended Models

| Model | Size | Notes |
|-------|------|-------|
| `llama3.2:1b` | ~1GB | Default, fast, good quality |
| `llama3.2:3b` | ~2GB | Better responses, still fast |
| `mistral:7b` | ~4GB | High quality, requires more RAM |

## Development

### Project Structure

```
TamaLLM/
├── cmd/tamallm/          # Main entry point
│   └── main.go
├── internal/
│   ├── game/             # Pure game engine
│   │   ├── state.go      # Pet state struct
│   │   ├── engine.go     # Game logic
│   │   └── engine_test.go
│   ├── tui/              # Terminal UI
│   │   ├── model.go      # Bubble Tea model
│   │   └── views.go      # View rendering
│   ├── llm/              # Ollama integration
│   │   ├── client.go     # API client
│   │   └── client_test.go
│   ├── storage/          # Save/load
│   │   ├── storage.go
│   │   └── storage_test.go
│   └── util/             # Helpers
│       ├── util.go
│       └── util_test.go
├── go.mod
├── go.sum
└── README.md
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/game/
```

### Building

```bash
# Build binary
go build -o tamallm ./cmd/tamallm

# Build with optimizations
go build -ldflags="-s -w" -o tamallm ./cmd/tamallm
```

## Architecture

### Bubble Tea Patterns

The TUI follows Bubble Tea best practices:
- Never blocks `Update()` with network I/O
- Uses `tea.Cmd` for async work (ticks, LLM calls, saves)
- Uses `tea.Tick` for simulation ticks

### LLM Integration

The LLM integration is:
- **Safe** - Tool calls are validated and bounded
- **Optional** - Game works fine without LLM
- **Deterministic** - LLM cannot directly change game state
- **Interface-driven** - Easy to mock for testing

### Tool Calling

When tool calling is enabled, the LLM can use these tools:

| Tool | Description | Permissions |
|------|-------------|-------------|
| `rand_int(min, max)` | Generate random number | Read-only |
| `propose_event(type, severity, desc)` | Propose game event | Validated by engine |
| `set_mood(mood, emoji, intensity)` | Set UI mood display | UI-only |
| `summarize_state()` | Get state summary | Read-only |

## Save File

Saves are stored in XDG-compliant locations:
- Linux: `~/.local/share/tamallm/tamallm_save.json`
- macOS: `~/Library/Application Support/tamallm/tamallm_save.json`

Save format is versioned JSON for forward compatibility.

## License

MIT License - see LICENSE file for details.

## Credits

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm
- LLM runtime by [Ollama](https://ollama.ai)
- Inspired by the classic Tamagotchi virtual pets
