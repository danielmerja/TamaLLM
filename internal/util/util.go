// Package util provides helper functions for the TamaLLM game.
package util

// Clamp constrains a value between min and max.
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampFloat constrains a float64 value between min and max.
func ClampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// RingBuffer is a fixed-size circular buffer for storing events.
type RingBuffer[T any] struct {
	data  []T
	head  int
	size  int
	cap   int
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, capacity),
		cap:  capacity,
	}
}

// Push adds an item to the buffer, overwriting the oldest if full.
func (rb *RingBuffer[T]) Push(item T) {
	rb.data[rb.head] = item
	rb.head = (rb.head + 1) % rb.cap
	if rb.size < rb.cap {
		rb.size++
	}
}

// Items returns all items in the buffer in chronological order.
func (rb *RingBuffer[T]) Items() []T {
	if rb.size == 0 {
		return nil
	}
	result := make([]T, rb.size)
	start := 0
	if rb.size == rb.cap {
		start = rb.head
	}
	for i := 0; i < rb.size; i++ {
		result[i] = rb.data[(start+i)%rb.cap]
	}
	return result
}

// Last returns the last n items (most recent first).
func (rb *RingBuffer[T]) Last(n int) []T {
	items := rb.Items()
	if n >= len(items) {
		// Reverse to get most recent first
		result := make([]T, len(items))
		for i := 0; i < len(items); i++ {
			result[i] = items[len(items)-1-i]
		}
		return result
	}
	result := make([]T, n)
	for i := 0; i < n; i++ {
		result[i] = items[len(items)-1-i]
	}
	return result
}

// Size returns the number of items in the buffer.
func (rb *RingBuffer[T]) Size() int {
	return rb.size
}

// Clear empties the buffer.
func (rb *RingBuffer[T]) Clear() {
	rb.head = 0
	rb.size = 0
}
