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
ollama pull qwen3:1.7b

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
| `a` | Toggle LLM auto mode |
| `?` | Toggle help |
| `d` | Toggle debug mode |
| `q` | Quit (auto-saves) |

### Debug Mode

Press `d` to toggle debug mode, which displays:
- **LLM Enabled/Pending/Auto status** - Shows if LLM is active and processing
- **TTS Enabled** - Shows if Text-to-Speech is active
- **Last LLM Input** - The action that triggered the LLM request
- **Last LLM Output** - The pet's message from the LLM
- **Age and Stage** - Pet's current age and life stage
- **Care Score** - Running average of care quality
- **Suggested Action** - (Auto mode only) The next recommended action

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
| Give Treat | +20 Happiness, +5 Hunger, high chance +1 Weight |
| Play | +25 Happiness, -20 Energy, -10 Hunger |
| Exercise | +15 Happiness, -25 Energy, -15 Hunger, +5 Health, +3 Discipline |
| Explore | +10 Happiness, -10 Energy, random encounters |
| Train | +8 Discipline, -15 Energy |
| Clean | +40 Hygiene, +5 Happiness |
| Sleep | Restores Energy over time |
| Medicine | Cures sickness, +20 Health |
| Praise | +10 Happiness, slight -2 Discipline |
| Scold | +10 Discipline, -15 Happiness |

### LLM Auto Mode

Press `a` or start with `--auto` to enable LLM-driven automatic care:
- The LLM will automatically perform actions when needed
- It uses the `get_suggested_action` tool to determine the best action
- Great for letting your pet care for itself while you watch!

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `qwen3:1.7b` | Model to use |
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
  --auto          Start with LLM auto mode enabled
  --tts           Enable Text-to-Speech (requires Supertonic-2)
  --tts-voice     TTS voice style (M1-M5, F1-F5, default: F3)
```

### Text-to-Speech (Supertonic-2)

TamaLLM supports text-to-speech using the [Supertonic-2](https://huggingface.co/Supertone/supertonic-2) model from Supertone. This allows your pet to "speak" its messages out loud!

#### Setup TTS

1. Install Python 3.8+ and pip
2. Install the Supertonic package:
   ```bash
   pip install supertonic sounddevice soundfile
   ```
   
   **Using uv (recommended):**
   ```bash
   uv venv
   uv pip install supertonic sounddevice soundfile
   ```

3. Run with TTS enabled:
   ```bash
   go run ./cmd/tamallm --tts
   ```
   
   TamaLLM automatically detects and uses `uv run` when packages are installed via `uv pip install` in a `uv venv` environment.

#### Voice Styles

| Style | Description |
|-------|-------------|
| M1-M5 | Male voices with varying tones |
| F1-F5 | Female voices with varying tones |

Example with custom voice:
```bash
go run ./cmd/tamallm --tts --tts-voice M2
```

### Recommended Models

| Model | Size | Notes |
|-------|------|-------|
| `qwen3:1.7b` | ~1.5GB | Default, fast, good quality |
| `qwen3:4b` | ~2.5GB | Better responses, still fast |
| `qwen3:8b` | ~5.2GB | High quality, requires more RAM |

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
│   ├── tts/              # Text-to-Speech (Supertonic-2)
│   │   ├── tts.go        # TTS client
│   │   └── tts_test.go
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
- **Action-driven** - LLM can request actions through validated tools
- **Interface-driven** - Easy to mock for testing

### Tool Calling

When tool calling is enabled, the LLM can use these tools:

| Tool | Description | Permissions |
|------|-------------|-------------|
| `rand_int(min, max)` | Generate random number | Read-only |
| `propose_event(type, severity, desc)` | Propose game event | Validated by engine |
| `set_mood(mood, emoji, intensity)` | Set UI mood display | UI-only |
| `summarize_state()` | Get state summary | Read-only |
| `request_action(action)` | Request to perform an action | Validated and executed |
| `get_valid_actions()` | Get list of currently valid actions | Read-only |
| `get_suggested_action()` | Get suggested action based on needs | Read-only |

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
