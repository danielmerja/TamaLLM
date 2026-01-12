// Package game implements the core game engine for TamaLLM.
package game

import (
	"math/rand"
	"time"
)

// Stage represents the pet's life stage.
type Stage string

const (
	StageEgg   Stage = "egg"
	StageBaby  Stage = "baby"
	StageChild Stage = "child"
	StageTeen  Stage = "teen"
	StageAdult Stage = "adult"
)

// AdultType represents the type of adult the pet evolved into.
type AdultType string

const (
	AdultTypeHealthy    AdultType = "healthy"
	AdultTypeChubby     AdultType = "chubby"
	AdultTypeNaughty    AdultType = "naughty"
	AdultTypeNegligence AdultType = "negligence"
)

// Personality traits for the pet.
type Personality struct {
	Boldness      int `json:"boldness"`      // 0=shy, 100=bold
	Independence  int `json:"independence"`  // 0=needy, 100=independent
	Playfulness   int `json:"playfulness"`   // 0=serious, 100=playful
}

// MemoryEvent represents a single event in the pet's memory.
type MemoryEvent struct {
	Time        time.Time `json:"time"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

// Alert types for the pet.
type AlertType string

const (
	AlertHungry   AlertType = "hungry"
	AlertDirty    AlertType = "dirty"
	AlertSick     AlertType = "sick"
	AlertTired    AlertType = "tired"
	AlertSad      AlertType = "sad"
	AlertCritical AlertType = "critical"
)

// State represents the complete pet state.
type State struct {
	// Identity
	Name      string    `json:"name"`
	Species   string    `json:"species"`
	Age       int64     `json:"age"` // in seconds
	Stage     Stage     `json:"stage"`
	AdultType AdultType `json:"adult_type,omitempty"`
	Alive     bool      `json:"alive"`
	CreatedAt time.Time `json:"created_at"`

	// Core stats (0-100)
	Hunger    int `json:"hunger"`    // 0=starving, 100=full
	Happiness int `json:"happiness"` // 0=miserable, 100=ecstatic
	Energy    int `json:"energy"`    // 0=exhausted, 100=energized
	Hygiene   int `json:"hygiene"`   // 0=filthy, 100=clean
	Health    int `json:"health"`    // 0=dead, 100=perfect

	// Other attributes
	Weight     int  `json:"weight"`     // normal is around 5-10
	Discipline int  `json:"discipline"` // 0=unruly, 100=well-behaved
	IsSick     bool `json:"is_sick"`
	IsSleeping bool `json:"is_sleeping"`

	// Personality (seeded at creation)
	Personality Personality `json:"personality"`

	// Memory
	Memory []MemoryEvent `json:"memory"`

	// Tracking for evolution
	TotalSicknessEvents int     `json:"total_sickness_events"`
	AverageCareScore    float64 `json:"average_care_score"`
	CareScoreCount      int     `json:"care_score_count"`

	// Mood (set by LLM for UI flair)
	Mood      string `json:"mood,omitempty"`
	MoodEmoji string `json:"mood_emoji,omitempty"`

	// Last message from pet
	LastMessage string `json:"last_message,omitempty"`

	// Random seed for deterministic behavior
	RandSeed int64 `json:"rand_seed"`
}

// NewState creates a new pet state with the given name.
func NewState(name, species string, seed int64) *State {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	return &State{
		Name:        name,
		Species:     species,
		Age:         0,
		Stage:       StageEgg,
		Alive:       true,
		CreatedAt:   time.Now(),
		Hunger:      80,
		Happiness:   80,
		Energy:      100,
		Hygiene:     100,
		Health:      100,
		Weight:      5,
		Discipline:  50,
		IsSick:      false,
		IsSleeping:  false,
		Personality: Personality{
			Boldness:     rng.Intn(101),
			Independence: rng.Intn(101),
			Playfulness:  rng.Intn(101),
		},
		Memory:           make([]MemoryEvent, 0, 20),
		RandSeed:         seed,
		Mood:             "neutral",
		MoodEmoji:        "😊",
		LastMessage:      "...",
	}
}

// AddMemory adds an event to the pet's memory, keeping only the last 20.
func (s *State) AddMemory(eventType, description string) {
	event := MemoryEvent{
		Time:        time.Now(),
		Type:        eventType,
		Description: description,
	}
	s.Memory = append(s.Memory, event)
	if len(s.Memory) > 20 {
		s.Memory = s.Memory[len(s.Memory)-20:]
	}
}

// GetAlerts returns the current alerts based on pet state.
func (s *State) GetAlerts() []AlertType {
	var alerts []AlertType
	if s.Hunger < 30 {
		alerts = append(alerts, AlertHungry)
	}
	if s.Hygiene < 30 {
		alerts = append(alerts, AlertDirty)
	}
	if s.IsSick {
		alerts = append(alerts, AlertSick)
	}
	if s.Energy < 20 {
		alerts = append(alerts, AlertTired)
	}
	if s.Happiness < 30 {
		alerts = append(alerts, AlertSad)
	}
	if s.Health < 30 {
		alerts = append(alerts, AlertCritical)
	}
	return alerts
}

// GetRecentMemory returns the last n memory events.
func (s *State) GetRecentMemory(n int) []MemoryEvent {
	if n >= len(s.Memory) {
		result := make([]MemoryEvent, len(s.Memory))
		copy(result, s.Memory)
		return result
	}
	return s.Memory[len(s.Memory)-n:]
}

// UpdateCareScore updates the running average of care quality.
func (s *State) UpdateCareScore() {
	// Calculate current care score based on stats
	score := float64(s.Hunger+s.Happiness+s.Energy+s.Hygiene+s.Health) / 5.0
	s.CareScoreCount++
	s.AverageCareScore = s.AverageCareScore + (score-s.AverageCareScore)/float64(s.CareScoreCount)
}
