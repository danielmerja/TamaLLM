package util

import (
	"testing"
)

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min, max, expected int
	}{
		{50, 0, 100, 50},  // Value within range
		{-10, 0, 100, 0},  // Value below min
		{150, 0, 100, 100}, // Value above max
		{0, 0, 100, 0},    // Value at min
		{100, 0, 100, 100}, // Value at max
	}

	for _, tt := range tests {
		result := Clamp(tt.value, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("Clamp(%d, %d, %d) = %d; want %d",
				tt.value, tt.min, tt.max, result, tt.expected)
		}
	}
}

func TestClampFloat(t *testing.T) {
	tests := []struct {
		value, min, max, expected float64
	}{
		{50.5, 0, 100, 50.5},
		{-10.5, 0, 100, 0},
		{150.5, 0, 100, 100},
	}

	for _, tt := range tests {
		result := ClampFloat(tt.value, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("ClampFloat(%f, %f, %f) = %f; want %f",
				tt.value, tt.min, tt.max, result, tt.expected)
		}
	}
}

func TestRingBuffer_Push(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	items := rb.Items()
	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	expected := []int{1, 2, 3}
	for i, v := range items {
		if v != expected[i] {
			t.Errorf("Item %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer[int](3)

	// Push 5 items into buffer of size 3
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	items := rb.Items()
	if len(items) != 3 {
		t.Errorf("Expected 3 items after overflow, got %d", len(items))
	}

	// Should have the last 3 items in order
	expected := []int{3, 4, 5}
	for i, v := range items {
		if v != expected[i] {
			t.Errorf("Item %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestRingBuffer_Last(t *testing.T) {
	rb := NewRingBuffer[int](5)

	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	last3 := rb.Last(3)
	if len(last3) != 3 {
		t.Errorf("Expected 3 items, got %d", len(last3))
	}

	// Most recent first
	expected := []int{5, 4, 3}
	for i, v := range last3 {
		if v != expected[i] {
			t.Errorf("Last item %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestRingBuffer_LastMoreThanSize(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)

	// Ask for more than we have
	last := rb.Last(5)
	if len(last) != 2 {
		t.Errorf("Expected 2 items, got %d", len(last))
	}
}

func TestRingBuffer_Size(t *testing.T) {
	rb := NewRingBuffer[int](5)

	if rb.Size() != 0 {
		t.Errorf("Expected size 0, got %d", rb.Size())
	}

	rb.Push(1)
	if rb.Size() != 1 {
		t.Errorf("Expected size 1, got %d", rb.Size())
	}

	rb.Push(2)
	rb.Push(3)
	if rb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", rb.Size())
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](5)

	rb.Push(1)
	rb.Push(2)
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", rb.Size())
	}

	items := rb.Items()
	if items != nil {
		t.Errorf("Expected nil items after clear, got %v", items)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer[string](3)

	items := rb.Items()
	if items != nil {
		t.Errorf("Expected nil for empty buffer, got %v", items)
	}

	last := rb.Last(5)
	if len(last) != 0 {
		t.Errorf("Expected empty last for empty buffer, got %v", last)
	}
}
