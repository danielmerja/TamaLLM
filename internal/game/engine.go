package game

import (
	"math/rand"
	"strconv"

	"github.com/danielmerja/TamaLLM/internal/util"
)

// Game balance constants
const (
	// Action probabilities
	ExerciseWeightLossChance = 0.4 // Chance to lose weight during exercise
	TreatWeightGainChance    = 0.7 // Chance to gain weight from treats
	SnackWeightGainChance    = 0.5 // Chance to gain weight from snacks
	PlayWeightLossChance     = 0.3 // Chance to lose weight during play
)

// Engine handles game logic and state transitions.
type Engine struct {
	State *State
	rng   *rand.Rand

	// Event cooldowns (in ticks)
	lastRandomEvent int64
	tickCount       int64
}

// NewEngine creates a new game engine with the given state.
func NewEngine(state *State) *Engine {
	return &Engine{
		State: state,
		rng:   rand.New(rand.NewSource(state.RandSeed)),
	}
}

// Action represents a player action.
type Action string

const (
	ActionFeedMeal   Action = "feed_meal"
	ActionFeedSnack  Action = "feed_snack"
	ActionPlay       Action = "play"
	ActionClean      Action = "clean"
	ActionSleep      Action = "sleep"
	ActionWake       Action = "wake"
	ActionMedicine   Action = "medicine"
	ActionScold      Action = "scold"
	ActionPraise     Action = "praise"
	ActionExercise   Action = "exercise"
	ActionExplore    Action = "explore"
	ActionTrain      Action = "train"
	ActionTreat      Action = "treat"
)

// PerformAction executes a player action and returns a description.
func (e *Engine) PerformAction(action Action) string {
	if !e.State.Alive {
		return "Your pet has passed away..."
	}

	switch action {
	case ActionFeedMeal:
		return e.feedMeal()
	case ActionFeedSnack:
		return e.feedSnack()
	case ActionPlay:
		return e.play()
	case ActionClean:
		return e.clean()
	case ActionSleep:
		return e.sleep()
	case ActionWake:
		return e.wake()
	case ActionMedicine:
		return e.medicine()
	case ActionScold:
		return e.scold()
	case ActionPraise:
		return e.praise()
	case ActionExercise:
		return e.exercise()
	case ActionExplore:
		return e.explore()
	case ActionTrain:
		return e.train()
	case ActionTreat:
		return e.treat()
	default:
		return "Unknown action"
	}
}

func (e *Engine) feedMeal() string {
	if e.State.IsSleeping {
		return "Can't feed while sleeping!"
	}
	if e.State.Hunger >= 90 {
		e.State.AddMemory("feed", "Refused food - too full")
		return "Not hungry right now!"
	}

	e.State.Hunger = util.Clamp(e.State.Hunger+30, 0, 100)
	e.State.Happiness = util.Clamp(e.State.Happiness+5, 0, 100)
	e.State.Weight = util.Clamp(e.State.Weight+1, 1, 20)
	e.State.AddMemory("feed", "Had a nice meal")
	return "Yummy! That was a good meal!"
}

func (e *Engine) feedSnack() string {
	if e.State.IsSleeping {
		return "Can't feed while sleeping!"
	}

	e.State.Hunger = util.Clamp(e.State.Hunger+10, 0, 100)
	e.State.Happiness = util.Clamp(e.State.Happiness+15, 0, 100)
	// Snacks risk weight gain
	if e.rng.Float64() < SnackWeightGainChance {
		e.State.Weight = util.Clamp(e.State.Weight+1, 1, 20)
	}
	e.State.AddMemory("snack", "Enjoyed a tasty snack")
	return "Snack time! Delicious!"
}

func (e *Engine) play() string {
	if e.State.IsSleeping {
		return "Can't play while sleeping!"
	}
	if e.State.Energy < 10 {
		e.State.AddMemory("play", "Too tired to play")
		return "Too tired to play..."
	}
	if e.State.IsSick {
		e.State.AddMemory("play", "Too sick to play")
		return "Don't feel well enough to play..."
	}

	e.State.Happiness = util.Clamp(e.State.Happiness+25, 0, 100)
	e.State.Energy = util.Clamp(e.State.Energy-20, 0, 100)
	e.State.Hunger = util.Clamp(e.State.Hunger-10, 0, 100)
	// Playing can reduce weight
	if e.State.Weight > 5 && e.rng.Float64() < PlayWeightLossChance {
		e.State.Weight--
	}
	e.State.AddMemory("play", "Had fun playing!")
	return "That was so much fun!"
}

func (e *Engine) clean() string {
	if e.State.IsSleeping {
		return "Can't clean while sleeping!"
	}

	e.State.Hygiene = util.Clamp(e.State.Hygiene+40, 0, 100)
	e.State.Happiness = util.Clamp(e.State.Happiness+5, 0, 100)
	e.State.AddMemory("clean", "Got a nice bath")
	return "Squeaky clean!"
}

func (e *Engine) sleep() string {
	if e.State.IsSleeping {
		return "Already sleeping!"
	}
	if e.State.Energy > 80 {
		e.State.AddMemory("sleep", "Not tired enough to sleep")
		return "Not sleepy yet!"
	}

	e.State.IsSleeping = true
	e.State.AddMemory("sleep", "Went to sleep")
	return "Zzz... *goes to sleep*"
}

func (e *Engine) wake() string {
	if !e.State.IsSleeping {
		return "Already awake!"
	}

	wasWellRested := e.State.Energy >= 70
	e.State.IsSleeping = false

	if !wasWellRested {
		// Waking early has consequences
		e.State.Happiness = util.Clamp(e.State.Happiness-10, 0, 100)
		e.State.AddMemory("wake", "Woke up too early, still tired")
		return "*yawn* Still sleepy..."
	}

	e.State.AddMemory("wake", "Woke up refreshed")
	return "*stretches* Good morning!"
}

func (e *Engine) medicine() string {
	if !e.State.IsSick {
		e.State.AddMemory("medicine", "Refused medicine - not sick")
		return "Don't need medicine right now!"
	}

	e.State.IsSick = false
	e.State.Health = util.Clamp(e.State.Health+20, 0, 100)
	e.State.Happiness = util.Clamp(e.State.Happiness-5, 0, 100) // Medicine tastes bad
	e.State.AddMemory("medicine", "Took medicine, feeling better")
	return "Yuck, but feeling better!"
}

func (e *Engine) scold() string {
	if e.State.IsSleeping {
		return "Can't scold while sleeping!"
	}

	// Scolding increases discipline but decreases happiness
	e.State.Discipline = util.Clamp(e.State.Discipline+10, 0, 100)
	e.State.Happiness = util.Clamp(e.State.Happiness-15, 0, 100)
	e.State.AddMemory("scold", "Got scolded")

	if e.State.Personality.Boldness > 70 {
		return "*pouts defiantly* Hmph!"
	}
	return "*looks down sadly* Sorry..."
}

func (e *Engine) praise() string {
	if e.State.IsSleeping {
		return "Can't praise while sleeping!"
	}

	e.State.Happiness = util.Clamp(e.State.Happiness+10, 0, 100)
	// Praise slightly decreases discipline if overdone
	if e.State.Discipline > 30 {
		e.State.Discipline = util.Clamp(e.State.Discipline-2, 0, 100)
	}
	e.State.AddMemory("praise", "Got praised")

	if e.State.Personality.Playfulness > 70 {
		return "*bounces happily* Yay!"
	}
	return "*smiles* Thank you!"
}

func (e *Engine) exercise() string {
	if e.State.IsSleeping {
		return "Can't exercise while sleeping!"
	}
	if e.State.Energy < 20 {
		e.State.AddMemory("exercise", "Too tired to exercise")
		return "Too tired to exercise..."
	}
	if e.State.IsSick {
		e.State.AddMemory("exercise", "Too sick to exercise")
		return "Don't feel well enough to exercise..."
	}

	e.State.Happiness = util.Clamp(e.State.Happiness+15, 0, 100)
	e.State.Energy = util.Clamp(e.State.Energy-25, 0, 100)
	e.State.Hunger = util.Clamp(e.State.Hunger-15, 0, 100)
	e.State.Health = util.Clamp(e.State.Health+5, 0, 100)
	e.State.Discipline = util.Clamp(e.State.Discipline+3, 0, 100)

	// Exercise helps lose weight
	if e.State.Weight > 5 && e.rng.Float64() < ExerciseWeightLossChance {
		e.State.Weight--
	}
	e.State.AddMemory("exercise", "Had a good workout!")

	if e.State.Personality.Boldness > 60 {
		return "*flexes proudly* Getting stronger!"
	}
	return "That was a good workout!"
}

func (e *Engine) explore() string {
	if e.State.IsSleeping {
		return "Can't explore while sleeping!"
	}
	if e.State.Energy < 15 {
		e.State.AddMemory("explore", "Too tired to explore")
		return "Too tired to explore..."
	}

	e.State.Happiness = util.Clamp(e.State.Happiness+10, 0, 100)
	e.State.Energy = util.Clamp(e.State.Energy-10, 0, 100)
	e.State.Hunger = util.Clamp(e.State.Hunger-5, 0, 100)

	// Random exploration outcomes
	roll := e.rng.Float64()
	if roll < 0.2 {
		// Found something good
		e.State.Happiness = util.Clamp(e.State.Happiness+10, 0, 100)
		e.State.AddMemory("explore", "Found something interesting!")
		return "*excited* Look what I found!"
	} else if roll < 0.35 {
		// Got a bit dirty
		e.State.Hygiene = util.Clamp(e.State.Hygiene-15, 0, 100)
		e.State.AddMemory("explore", "Got dirty while exploring")
		return "*covered in dust* Oops, got a bit messy..."
	} else if roll < 0.45 {
		// Small scare
		if e.State.Personality.Boldness < 40 {
			e.State.Happiness = util.Clamp(e.State.Happiness-5, 0, 100)
			e.State.AddMemory("explore", "Got startled while exploring")
			return "*jumps back* That was scary!"
		}
		e.State.AddMemory("explore", "Had an adventure!")
		return "What an adventure!"
	}

	e.State.AddMemory("explore", "Explored around")
	if e.State.Personality.Independence > 60 {
		return "*wanders happily* So much to see!"
	}
	return "That was fun exploring!"
}

func (e *Engine) train() string {
	if e.State.IsSleeping {
		return "Can't train while sleeping!"
	}
	if e.State.Energy < 15 {
		e.State.AddMemory("train", "Too tired to train")
		return "Too tired to train..."
	}

	e.State.Energy = util.Clamp(e.State.Energy-15, 0, 100)
	e.State.Discipline = util.Clamp(e.State.Discipline+8, 0, 100)

	// Training success based on discipline
	if e.State.Discipline > 50 || e.rng.Float64() < 0.6 {
		e.State.Happiness = util.Clamp(e.State.Happiness+5, 0, 100)
		e.State.AddMemory("train", "Learned something new!")
		return "*proud* I did it! I learned something!"
	}

	e.State.Happiness = util.Clamp(e.State.Happiness-3, 0, 100)
	e.State.AddMemory("train", "Training was tough")
	return "*struggles* Training is hard..."
}

func (e *Engine) treat() string {
	if e.State.IsSleeping {
		return "Can't give treats while sleeping!"
	}

	e.State.Happiness = util.Clamp(e.State.Happiness+20, 0, 100)
	e.State.Hunger = util.Clamp(e.State.Hunger+5, 0, 100)

	// Treats always risk weight gain
	if e.rng.Float64() < TreatWeightGainChance {
		e.State.Weight = util.Clamp(e.State.Weight+1, 1, 20)
	}
	e.State.AddMemory("treat", "Got a special treat!")

	if e.State.Personality.Playfulness > 60 {
		return "*does a happy dance* Best day ever!"
	}
	return "*munches happily* Yummy treat!"
}

// Tick advances the game state by one tick (typically 1 second).
func (e *Engine) Tick() {
	if !e.State.Alive {
		return
	}

	e.tickCount++
	e.State.Age++

	// Handle sleeping - restore energy
	if e.State.IsSleeping {
		e.State.Energy = util.Clamp(e.State.Energy+5, 0, 100)
		// Auto-wake when fully rested
		if e.State.Energy >= 100 {
			e.State.IsSleeping = false
			e.State.AddMemory("wake", "Woke up fully rested")
		}
	} else {
		// Stat decay when awake
		e.decayStats()
	}

	// Check for sickness
	e.checkSickness()

	// Check for death
	e.checkDeath()

	// Check for evolution
	e.checkEvolution()

	// Random events
	e.checkRandomEvents()

	// Update care score periodically
	if e.tickCount%10 == 0 {
		e.State.UpdateCareScore()
	}
}

func (e *Engine) decayStats() {
	// Base decay rates per tick
	e.State.Hunger = util.Clamp(e.State.Hunger-1, 0, 100)
	e.State.Energy = util.Clamp(e.State.Energy-1, 0, 100)

	// Slower decay for hygiene
	if e.tickCount%3 == 0 {
		e.State.Hygiene = util.Clamp(e.State.Hygiene-1, 0, 100)
	}

	// Happiness decays faster if other stats are low
	happinessDecay := 0
	if e.State.Hunger < 30 {
		happinessDecay++
	}
	if e.State.Energy < 30 {
		happinessDecay++
	}
	if e.State.Hygiene < 30 {
		happinessDecay++
	}
	if e.State.IsSick {
		happinessDecay += 2
	}
	if happinessDecay > 0 && e.tickCount%2 == 0 {
		e.State.Happiness = util.Clamp(e.State.Happiness-happinessDecay, 0, 100)
	}

	// Health degrades when sick or stats are critically low
	if e.State.IsSick {
		e.State.Health = util.Clamp(e.State.Health-1, 0, 100)
	}
	if e.State.Hunger == 0 || e.State.Energy == 0 {
		e.State.Health = util.Clamp(e.State.Health-2, 0, 100)
	}
}

func (e *Engine) checkSickness() {
	if e.State.IsSick {
		return
	}

	// Chance of getting sick increases with poor hygiene and low stats
	sickChance := 0.0
	if e.State.Hygiene < 20 {
		sickChance += 0.02
	}
	if e.State.Hunger < 20 {
		sickChance += 0.01
	}
	if e.State.Health < 50 {
		sickChance += 0.01
	}

	if sickChance > 0 && e.rng.Float64() < sickChance {
		e.State.IsSick = true
		e.State.TotalSicknessEvents++
		e.State.AddMemory("sick", "Got sick!")
	}
}

func (e *Engine) checkDeath() {
	if e.State.Health <= 0 {
		e.State.Alive = false
		e.State.AddMemory("death", "Passed away due to poor health")
		return
	}

	// Death from prolonged neglect (all core stats critically low)
	if e.State.Hunger < 10 && e.State.Happiness < 10 && e.State.Energy < 10 {
		e.State.Alive = false
		e.State.AddMemory("death", "Passed away from neglect")
	}
}

// Evolution thresholds (in seconds/age)
const (
	AgeEggToHatch   = 30   // 30 seconds
	AgeBabyToChild  = 120  // 2 minutes
	AgeChildToTeen  = 300  // 5 minutes
	AgeTeenToAdult  = 600  // 10 minutes
)

func (e *Engine) checkEvolution() {
	currentStage := e.State.Stage
	age := e.State.Age

	switch currentStage {
	case StageEgg:
		if age >= AgeEggToHatch {
			e.State.Stage = StageBaby
			e.State.AddMemory("evolution", "Hatched into a baby!")
		}
	case StageBaby:
		if age >= AgeBabyToChild {
			e.State.Stage = StageChild
			e.State.AddMemory("evolution", "Grew into a child!")
		}
	case StageChild:
		if age >= AgeChildToTeen {
			e.State.Stage = StageTeen
			e.State.AddMemory("evolution", "Became a teenager!")
		}
	case StageTeen:
		if age >= AgeTeenToAdult {
			e.evolveToAdult()
		}
	}
}

func (e *Engine) evolveToAdult() {
	e.State.Stage = StageAdult

	// Determine adult type based on care quality
	avgCare := e.State.AverageCareScore
	discipline := e.State.Discipline
	sickness := e.State.TotalSicknessEvents
	weight := e.State.Weight

	// Decision tree for adult type
	if avgCare >= 70 && sickness <= 2 && discipline >= 50 {
		e.State.AdultType = AdultTypeHealthy
		e.State.AddMemory("evolution", "Evolved into a healthy adult!")
	} else if weight >= 12 {
		e.State.AdultType = AdultTypeChubby
		e.State.AddMemory("evolution", "Evolved into a chubby adult!")
	} else if discipline < 30 {
		e.State.AdultType = AdultTypeNaughty
		e.State.AddMemory("evolution", "Evolved into a naughty adult!")
	} else {
		e.State.AdultType = AdultTypeNegligence
		e.State.AddMemory("evolution", "Evolved into an adult (some neglect)")
	}
}

func (e *Engine) checkRandomEvents() {
	// Don't trigger events too frequently
	if e.tickCount-e.lastRandomEvent < 30 {
		return
	}

	// Small chance of random event
	if e.rng.Float64() > 0.05 {
		return
	}

	e.lastRandomEvent = e.tickCount

	// Pick a random event based on stage
	events := []struct {
		chance float64
		apply  func()
	}{
		{0.15, func() {
			e.State.Happiness = util.Clamp(e.State.Happiness+5, 0, 100)
			e.State.AddMemory("random", "Found something fun!")
		}},
		{0.10, func() {
			e.State.Hunger = util.Clamp(e.State.Hunger-5, 0, 100)
			e.State.AddMemory("random", "Got extra hungry from activity")
		}},
		{0.10, func() {
			e.State.Hygiene = util.Clamp(e.State.Hygiene-10, 0, 100)
			e.State.AddMemory("random", "Made a mess playing")
		}},
		{0.10, func() {
			if e.State.Stage != StageEgg && e.State.Personality.Playfulness > 50 {
				e.State.Happiness = util.Clamp(e.State.Happiness+10, 0, 100)
				e.State.AddMemory("random", "Had a burst of joy!")
			}
		}},
		{0.08, func() {
			if e.State.Personality.Independence < 30 {
				e.State.Happiness = util.Clamp(e.State.Happiness-5, 0, 100)
				e.State.AddMemory("random", "Felt lonely")
			}
		}},
		// New events
		{0.08, func() {
			if e.State.Stage != StageEgg && !e.State.IsSleeping {
				e.State.Energy = util.Clamp(e.State.Energy+5, 0, 100)
				e.State.AddMemory("random", "Took a refreshing nap")
			}
		}},
		{0.07, func() {
			if e.State.Stage == StageAdult || e.State.Stage == StageTeen {
				e.State.Discipline = util.Clamp(e.State.Discipline+3, 0, 100)
				e.State.AddMemory("random", "Had a moment of self-reflection")
			}
		}},
		{0.06, func() {
			if e.State.Stage != StageEgg && e.State.Hygiene > 50 {
				e.State.Hygiene = util.Clamp(e.State.Hygiene-20, 0, 100)
				e.State.AddMemory("random", "Rolled in something stinky!")
			}
		}},
		{0.06, func() {
			if e.State.Stage != StageEgg && e.State.Personality.Boldness > 60 {
				e.State.Happiness = util.Clamp(e.State.Happiness+8, 0, 100)
				e.State.Energy = util.Clamp(e.State.Energy-5, 0, 100)
				e.State.AddMemory("random", "Had an exciting adventure!")
			}
		}},
		{0.05, func() {
			if e.State.Stage != StageEgg && e.State.Health > 70 {
				e.State.Health = util.Clamp(e.State.Health+3, 0, 100)
				e.State.AddMemory("random", "Feeling extra healthy today!")
			}
		}},
		{0.05, func() {
			if e.State.Stage == StageChild || e.State.Stage == StageTeen {
				e.State.Discipline = util.Clamp(e.State.Discipline-3, 0, 100)
				e.State.AddMemory("random", "Got into a bit of mischief")
			}
		}},
		{0.05, func() {
			if !e.State.IsSleeping && e.State.Happiness > 60 {
				e.State.Happiness = util.Clamp(e.State.Happiness+5, 0, 100)
				e.State.AddMemory("random", "Remembered a happy moment")
			}
		}},
		{0.05, func() {
			if e.State.Stage != StageEgg && e.State.Energy < 70 {
				e.State.Energy = util.Clamp(e.State.Energy+10, 0, 100)
				e.State.AddMemory("random", "Got a second wind!")
			}
		}},
	}

	roll := e.rng.Float64()
	cumulative := 0.0
	for _, event := range events {
		cumulative += event.chance
		if roll < cumulative {
			event.apply()
			break
		}
	}
}

// ProposeEvent allows external systems (like LLM) to propose events.
// Returns whether the event was accepted and a reason.
func (e *Engine) ProposeEvent(eventType string, severity int, description string) (bool, string) {
	// Validate and apply cooldowns
	if e.tickCount-e.lastRandomEvent < 20 {
		return false, "event cooldown active"
	}

	// Severity validation (1-5)
	if severity < 1 || severity > 5 {
		return false, "invalid severity"
	}

	// Don't accept high-severity events from external sources
	if severity > 3 {
		return false, "severity too high for external events"
	}

	// Apply the event based on type
	switch eventType {
	case "happiness_boost":
		e.State.Happiness = util.Clamp(e.State.Happiness+severity*3, 0, 100)
		e.State.AddMemory("proposed", description)
		e.lastRandomEvent = e.tickCount
		return true, "event accepted"
	case "energy_boost":
		e.State.Energy = util.Clamp(e.State.Energy+severity*2, 0, 100)
		e.State.AddMemory("proposed", description)
		e.lastRandomEvent = e.tickCount
		return true, "event accepted"
	case "hunger_spike":
		e.State.Hunger = util.Clamp(e.State.Hunger-severity*3, 0, 100)
		e.State.AddMemory("proposed", description)
		e.lastRandomEvent = e.tickCount
		return true, "event accepted"
	default:
		return false, "unknown event type"
	}
}

// SetMood allows external systems to set the pet's mood (UI flair only).
func (e *Engine) SetMood(mood, emoji string, intensity int) bool {
	if intensity < 1 || intensity > 5 {
		return false
	}
	e.State.Mood = mood
	e.State.MoodEmoji = emoji
	return true
}

// SummarizeState returns a compact summary for LLM context.
func (e *Engine) SummarizeState() string {
	s := e.State
	status := "awake"
	if s.IsSleeping {
		status = "sleeping"
	}
	healthStatus := "healthy"
	if s.IsSick {
		healthStatus = "sick"
	}

	summary := s.Name + " (" + string(s.Stage) + ", " + status + ", " + healthStatus + "): "
	summary += "hunger=" + strconv.Itoa(s.Hunger) + ", happy=" + strconv.Itoa(s.Happiness)
	summary += ", energy=" + strconv.Itoa(s.Energy) + ", hygiene=" + strconv.Itoa(s.Hygiene)
	summary += ", health=" + strconv.Itoa(s.Health) + ", discipline=" + strconv.Itoa(s.Discipline)

	alerts := s.GetAlerts()
	if len(alerts) > 0 {
		summary += " [ALERTS: "
		for i, a := range alerts {
			if i > 0 {
				summary += ", "
			}
			summary += string(a)
		}
		summary += "]"
	}

	return summary
}

// RandInt generates a random integer in the range [min, max].
func (e *Engine) RandInt(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return min + e.rng.Intn(max-min+1)
}

// ValidActions returns a list of currently valid actions the pet can perform.
func (e *Engine) ValidActions() []string {
	if !e.State.Alive {
		return nil
	}

	actions := []string{}

	if !e.State.IsSleeping {
		actions = append(actions, "feed_meal", "feed_snack", "play", "clean", "praise", "scold", "exercise", "explore", "train", "treat")
		if e.State.Energy <= 80 {
			actions = append(actions, "sleep")
		}
		if e.State.IsSick {
			actions = append(actions, "medicine")
		}
	} else {
		actions = append(actions, "wake")
	}

	return actions
}

// RequestAction allows the LLM to request an action be performed.
// Returns success status, result message, and whether the action was valid.
func (e *Engine) RequestAction(actionName string) (bool, string) {
	if !e.State.Alive {
		return false, "pet is not alive"
	}

	// Map string to Action type
	var action Action
	switch actionName {
	case "feed_meal":
		action = ActionFeedMeal
	case "feed_snack":
		action = ActionFeedSnack
	case "play":
		action = ActionPlay
	case "clean":
		action = ActionClean
	case "sleep":
		action = ActionSleep
	case "wake":
		action = ActionWake
	case "medicine":
		action = ActionMedicine
	case "scold":
		action = ActionScold
	case "praise":
		action = ActionPraise
	case "exercise":
		action = ActionExercise
	case "explore":
		action = ActionExplore
	case "train":
		action = ActionTrain
	case "treat":
		action = ActionTreat
	default:
		return false, "unknown action: " + actionName
	}

	// Perform the action
	result := e.PerformAction(action)
	return true, result
}

// GetSuggestedAction returns a suggested action based on current state.
// This helps the LLM make better decisions.
func (e *Engine) GetSuggestedAction() string {
	if !e.State.Alive {
		return ""
	}

	// Priority-based suggestions
	if e.State.IsSick {
		return "medicine"
	}
	if e.State.Hunger < 25 {
		return "feed_meal"
	}
	if e.State.Energy < 20 && !e.State.IsSleeping {
		return "sleep"
	}
	if e.State.IsSleeping && e.State.Energy >= 70 {
		return "wake"
	}
	if e.State.Hygiene < 30 {
		return "clean"
	}
	if e.State.Happiness < 30 && e.State.Energy >= 30 {
		return "play"
	}
	if e.State.Health < 50 && e.State.Energy >= 30 {
		return "exercise"
	}
	if e.State.Discipline < 40 && e.State.Energy >= 20 {
		return "train"
	}

	return ""
}
