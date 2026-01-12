package game

import (
	"testing"
)

func TestNewState(t *testing.T) {
	state := NewState("TestPet", "blob", 12345)

	if state.Name != "TestPet" {
		t.Errorf("Expected name 'TestPet', got '%s'", state.Name)
	}
	if state.Species != "blob" {
		t.Errorf("Expected species 'blob', got '%s'", state.Species)
	}
	if state.Stage != StageEgg {
		t.Errorf("Expected stage 'egg', got '%s'", state.Stage)
	}
	if !state.Alive {
		t.Error("Expected pet to be alive")
	}
	if state.RandSeed != 12345 {
		t.Errorf("Expected seed 12345, got %d", state.RandSeed)
	}

	// Check initial stats
	if state.Hunger != 80 {
		t.Errorf("Expected hunger 80, got %d", state.Hunger)
	}
	if state.Health != 100 {
		t.Errorf("Expected health 100, got %d", state.Health)
	}
}

func TestAddMemory(t *testing.T) {
	state := NewState("Test", "blob", 0)

	// Add 25 events
	for i := 0; i < 25; i++ {
		state.AddMemory("test", "Event")
	}

	// Should only keep last 20
	if len(state.Memory) != 20 {
		t.Errorf("Expected 20 memory events, got %d", len(state.Memory))
	}
}

func TestGetAlerts(t *testing.T) {
	state := NewState("Test", "blob", 0)

	// No alerts initially
	alerts := state.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected no alerts, got %d", len(alerts))
	}

	// Set low hunger
	state.Hunger = 20
	alerts = state.GetAlerts()
	found := false
	for _, a := range alerts {
		if a == AlertHungry {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected hungry alert")
	}

	// Set sick
	state.IsSick = true
	alerts = state.GetAlerts()
	found = false
	for _, a := range alerts {
		if a == AlertSick {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected sick alert")
	}
}

func TestEngine_Tick(t *testing.T) {
	state := NewState("Test", "blob", 12345)
	engine := NewEngine(state)

	initialHunger := state.Hunger
	initialEnergy := state.Energy

	// Run a tick
	engine.Tick()

	// Stats should decay
	if state.Hunger >= initialHunger {
		t.Error("Expected hunger to decrease")
	}
	if state.Energy >= initialEnergy {
		t.Error("Expected energy to decrease")
	}
	if state.Age != 1 {
		t.Errorf("Expected age 1, got %d", state.Age)
	}
}

func TestEngine_FeedMeal(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Hunger = 50
	engine := NewEngine(state)

	result := engine.PerformAction(ActionFeedMeal)

	if state.Hunger != 80 {
		t.Errorf("Expected hunger 80, got %d", state.Hunger)
	}
	if result == "" {
		t.Error("Expected non-empty result message")
	}
}

func TestEngine_FeedSnack(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Hunger = 50
	state.Happiness = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionFeedSnack)

	if state.Hunger != 60 {
		t.Errorf("Expected hunger 60, got %d", state.Hunger)
	}
	if state.Happiness != 65 {
		t.Errorf("Expected happiness 65, got %d", state.Happiness)
	}
}

func TestEngine_Play(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Happiness = 50
	state.Energy = 50
	state.Hunger = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionPlay)

	if state.Happiness != 75 {
		t.Errorf("Expected happiness 75, got %d", state.Happiness)
	}
	if state.Energy != 30 {
		t.Errorf("Expected energy 30, got %d", state.Energy)
	}
	if state.Hunger != 40 {
		t.Errorf("Expected hunger 40, got %d", state.Hunger)
	}
}

func TestEngine_PlayWhenTired(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Energy = 5
	engine := NewEngine(state)

	result := engine.PerformAction(ActionPlay)

	if result != "Too tired to play..." {
		t.Errorf("Expected tired message, got '%s'", result)
	}
}

func TestEngine_Clean(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Hygiene = 40
	engine := NewEngine(state)

	engine.PerformAction(ActionClean)

	if state.Hygiene != 80 {
		t.Errorf("Expected hygiene 80, got %d", state.Hygiene)
	}
}

func TestEngine_Sleep(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Energy = 30
	engine := NewEngine(state)

	engine.PerformAction(ActionSleep)

	if !state.IsSleeping {
		t.Error("Expected pet to be sleeping")
	}
}

func TestEngine_SleepNotTired(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Energy = 90
	engine := NewEngine(state)

	result := engine.PerformAction(ActionSleep)

	if state.IsSleeping {
		t.Error("Pet should not be sleeping when not tired")
	}
	if result != "Not sleepy yet!" {
		t.Errorf("Expected not sleepy message, got '%s'", result)
	}
}

func TestEngine_Wake(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSleeping = true
	state.Energy = 40
	engine := NewEngine(state)

	engine.PerformAction(ActionWake)

	if state.IsSleeping {
		t.Error("Pet should be awake")
	}
}

func TestEngine_WakeEarly(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSleeping = true
	state.Energy = 30
	state.Happiness = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionWake)

	// Happiness should decrease when waking early
	if state.Happiness != 40 {
		t.Errorf("Expected happiness 40, got %d", state.Happiness)
	}
}

func TestEngine_Medicine(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSick = true
	state.Health = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionMedicine)

	if state.IsSick {
		t.Error("Pet should no longer be sick")
	}
	if state.Health != 70 {
		t.Errorf("Expected health 70, got %d", state.Health)
	}
}

func TestEngine_MedicineNotSick(t *testing.T) {
	state := NewState("Test", "blob", 0)
	engine := NewEngine(state)

	result := engine.PerformAction(ActionMedicine)

	if result != "Don't need medicine right now!" {
		t.Errorf("Expected not sick message, got '%s'", result)
	}
}

func TestEngine_Scold(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Discipline = 50
	state.Happiness = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionScold)

	if state.Discipline != 60 {
		t.Errorf("Expected discipline 60, got %d", state.Discipline)
	}
	if state.Happiness != 35 {
		t.Errorf("Expected happiness 35, got %d", state.Happiness)
	}
}

func TestEngine_Praise(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Happiness = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionPraise)

	if state.Happiness != 60 {
		t.Errorf("Expected happiness 60, got %d", state.Happiness)
	}
}

func TestEngine_Evolution(t *testing.T) {
	state := NewState("Test", "blob", 12345)
	engine := NewEngine(state)

	// Should start as egg
	if state.Stage != StageEgg {
		t.Errorf("Expected egg, got %s", state.Stage)
	}

	// Advance to baby (30 ticks)
	for i := 0; i < 35; i++ {
		engine.Tick()
	}

	if state.Stage != StageBaby {
		t.Errorf("Expected baby after 35 ticks, got %s", state.Stage)
	}
}

func TestEngine_DeathFromHealth(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Health = 1
	state.IsSick = true
	engine := NewEngine(state)

	// Run ticks until health depletes
	for i := 0; i < 5; i++ {
		engine.Tick()
		if !state.Alive {
			break
		}
	}

	if state.Alive {
		t.Error("Pet should be dead from health depletion")
	}
}

func TestEngine_DeathFromNeglect(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Hunger = 5
	state.Happiness = 5
	state.Energy = 5
	engine := NewEngine(state)

	engine.Tick()

	if state.Alive {
		t.Error("Pet should be dead from neglect")
	}
}

func TestEngine_SleepingRestoresEnergy(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSleeping = true
	state.Energy = 50
	engine := NewEngine(state)

	engine.Tick()

	if state.Energy != 55 {
		t.Errorf("Expected energy 55 after sleeping tick, got %d", state.Energy)
	}
}

func TestEngine_AutoWakeWhenRested(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSleeping = true
	state.Energy = 96
	engine := NewEngine(state)

	engine.Tick()

	if state.IsSleeping {
		t.Error("Pet should auto-wake when fully rested")
	}
}

func TestEngine_ActionsWhileSleeping(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSleeping = true
	engine := NewEngine(state)

	actions := []Action{ActionFeedMeal, ActionFeedSnack, ActionPlay, ActionClean, ActionScold, ActionPraise}

	for _, action := range actions {
		result := engine.PerformAction(action)
		if result != "Can't feed while sleeping!" && result != "Can't play while sleeping!" &&
			result != "Can't clean while sleeping!" && result != "Can't scold while sleeping!" &&
			result != "Can't praise while sleeping!" {
			t.Errorf("Expected sleeping restriction for %s, got '%s'", action, result)
		}
	}
}

func TestEngine_ProposeEvent(t *testing.T) {
	state := NewState("Test", "blob", 12345)
	engine := NewEngine(state)
	state.Happiness = 50

	// Should accept valid events after cooldown
	engine.tickCount = 100

	accepted, reason := engine.ProposeEvent("happiness_boost", 2, "Feeling great!")
	if !accepted {
		t.Errorf("Expected event to be accepted, got: %s", reason)
	}

	if state.Happiness != 56 { // 50 + 2*3
		t.Errorf("Expected happiness 56, got %d", state.Happiness)
	}
}

func TestEngine_ProposeEventCooldown(t *testing.T) {
	state := NewState("Test", "blob", 0)
	engine := NewEngine(state)

	// Try immediately (no cooldown)
	accepted, reason := engine.ProposeEvent("happiness_boost", 1, "Test")

	if accepted {
		t.Error("Expected event to be rejected due to cooldown")
	}
	if reason != "event cooldown active" {
		t.Errorf("Expected cooldown reason, got: %s", reason)
	}
}

func TestEngine_ProposeEventHighSeverity(t *testing.T) {
	state := NewState("Test", "blob", 0)
	engine := NewEngine(state)
	engine.tickCount = 100

	accepted, reason := engine.ProposeEvent("happiness_boost", 5, "Test")

	if accepted {
		t.Error("Expected high severity event to be rejected")
	}
	if reason != "severity too high for external events" {
		t.Errorf("Expected severity reason, got: %s", reason)
	}
}

func TestEngine_SetMood(t *testing.T) {
	state := NewState("Test", "blob", 0)
	engine := NewEngine(state)

	ok := engine.SetMood("happy", "😊", 3)
	if !ok {
		t.Error("Expected SetMood to succeed")
	}
	if state.Mood != "happy" {
		t.Errorf("Expected mood 'happy', got '%s'", state.Mood)
	}
	if state.MoodEmoji != "😊" {
		t.Errorf("Expected emoji '😊', got '%s'", state.MoodEmoji)
	}
}

func TestEngine_SetMoodInvalidIntensity(t *testing.T) {
	state := NewState("Test", "blob", 0)
	engine := NewEngine(state)

	ok := engine.SetMood("happy", "😊", 10)
	if ok {
		t.Error("Expected SetMood to fail with invalid intensity")
	}
}

func TestEngine_SummarizeState(t *testing.T) {
	state := NewState("Tama", "blob", 0)
	engine := NewEngine(state)

	summary := engine.SummarizeState()

	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	if len(summary) > 500 {
		t.Error("Summary should be compact")
	}
}

func TestEngine_RandInt(t *testing.T) {
	state := NewState("Test", "blob", 12345)
	engine := NewEngine(state)

	// Should be deterministic with same seed
	results := make([]int, 5)
	for i := 0; i < 5; i++ {
		results[i] = engine.RandInt(1, 100)
	}

	// With seed 12345, should be reproducible
	state2 := NewState("Test2", "blob", 12345)
	engine2 := NewEngine(state2)

	for i := 0; i < 5; i++ {
		r := engine2.RandInt(1, 100)
		if r != results[i] {
			t.Errorf("Expected deterministic result %d at index %d, got %d", results[i], i, r)
		}
	}
}

func TestEngine_FullEvolutionCycle(t *testing.T) {
	state := NewState("Test", "blob", 12345)
	engine := NewEngine(state)

	// Keep stats healthy during evolution
	keepHealthy := func() {
		if state.Hunger < 50 {
			engine.PerformAction(ActionFeedMeal)
		}
		if state.Energy < 30 {
			engine.PerformAction(ActionSleep)
		}
		if state.Hygiene < 50 {
			engine.PerformAction(ActionClean)
		}
	}

	stages := []Stage{StageEgg, StageBaby, StageChild, StageTeen, StageAdult}
	stageIndex := 0

	// Run until adult or dead
	for i := 0; i < 700 && state.Alive && stageIndex < len(stages)-1; i++ {
		engine.Tick()
		keepHealthy()

		if state.Stage != stages[stageIndex] {
			stageIndex++
			t.Logf("Evolved to %s at tick %d", state.Stage, i)
		}
	}

	if !state.Alive {
		t.Error("Pet died during evolution test")
	}

	if state.Stage != StageAdult {
		t.Errorf("Expected adult stage, got %s", state.Stage)
	}
}

func TestEngine_Exercise(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Happiness = 50
	state.Energy = 50
	state.Hunger = 50
	state.Health = 50
	state.Discipline = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionExercise)

	if state.Happiness != 65 { // +15
		t.Errorf("Expected happiness 65, got %d", state.Happiness)
	}
	if state.Energy != 25 { // -25
		t.Errorf("Expected energy 25, got %d", state.Energy)
	}
	if state.Hunger != 35 { // -15
		t.Errorf("Expected hunger 35, got %d", state.Hunger)
	}
	if state.Health != 55 { // +5
		t.Errorf("Expected health 55, got %d", state.Health)
	}
	if state.Discipline != 53 { // +3
		t.Errorf("Expected discipline 53, got %d", state.Discipline)
	}
}

func TestEngine_ExerciseWhenTired(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Energy = 10
	engine := NewEngine(state)

	result := engine.PerformAction(ActionExercise)

	if result != "Too tired to exercise..." {
		t.Errorf("Expected tired message, got '%s'", result)
	}
}

func TestEngine_Explore(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Happiness = 50
	state.Energy = 50
	state.Hunger = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionExplore)

	if state.Happiness < 50 { // Should increase or stay same
		t.Errorf("Expected happiness >= 50, got %d", state.Happiness)
	}
	if state.Energy != 40 { // -10
		t.Errorf("Expected energy 40, got %d", state.Energy)
	}
}

func TestEngine_Train(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Energy = 50
	state.Discipline = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionTrain)

	if state.Energy != 35 { // -15
		t.Errorf("Expected energy 35, got %d", state.Energy)
	}
	if state.Discipline != 58 { // +8
		t.Errorf("Expected discipline 58, got %d", state.Discipline)
	}
}

func TestEngine_Treat(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Happiness = 50
	state.Hunger = 50
	engine := NewEngine(state)

	engine.PerformAction(ActionTreat)

	if state.Happiness != 70 { // +20
		t.Errorf("Expected happiness 70, got %d", state.Happiness)
	}
	if state.Hunger != 55 { // +5
		t.Errorf("Expected hunger 55, got %d", state.Hunger)
	}
}

func TestEngine_ValidActions(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Energy = 50
	engine := NewEngine(state)

	actions := engine.ValidActions()

	// Should include basic actions
	expected := map[string]bool{
		"feed_meal": true, "feed_snack": true, "play": true, "clean": true,
		"praise": true, "scold": true, "exercise": true, "explore": true,
		"train": true, "treat": true, "sleep": true,
	}

	for _, action := range actions {
		if !expected[action] && action != "medicine" && action != "wake" {
			t.Errorf("Unexpected action: %s", action)
		}
	}
}

func TestEngine_ValidActionsWhileSleeping(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSleeping = true
	engine := NewEngine(state)

	actions := engine.ValidActions()

	// Should only have wake
	if len(actions) != 1 || actions[0] != "wake" {
		t.Errorf("Expected only wake action while sleeping, got: %v", actions)
	}
}

func TestEngine_RequestAction(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Hunger = 50
	engine := NewEngine(state)

	success, result := engine.RequestAction("feed_meal")

	if !success {
		t.Error("Expected success")
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}
	if state.Hunger != 80 { // +30
		t.Errorf("Expected hunger 80, got %d", state.Hunger)
	}
}

func TestEngine_RequestActionUnknown(t *testing.T) {
	state := NewState("Test", "blob", 0)
	engine := NewEngine(state)

	success, result := engine.RequestAction("unknown_action")

	if success {
		t.Error("Expected failure for unknown action")
	}
	if result == "" {
		t.Error("Expected error message")
	}
}

func TestEngine_GetSuggestedAction(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.IsSick = true
	engine := NewEngine(state)

	suggested := engine.GetSuggestedAction()

	if suggested != "medicine" {
		t.Errorf("Expected medicine suggestion when sick, got: %s", suggested)
	}
}

func TestEngine_GetSuggestedActionHungry(t *testing.T) {
	state := NewState("Test", "blob", 0)
	state.Hunger = 20
	engine := NewEngine(state)

	suggested := engine.GetSuggestedAction()

	if suggested != "feed_meal" {
		t.Errorf("Expected feed_meal suggestion when hungry, got: %s", suggested)
	}
}

func TestEngine_GetSuggestedActionNone(t *testing.T) {
	state := NewState("Test", "blob", 0)
	// Default state has good stats
	engine := NewEngine(state)

	suggested := engine.GetSuggestedAction()

	if suggested != "" {
		t.Errorf("Expected no suggestion with good stats, got: %s", suggested)
	}
}
