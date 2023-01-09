package util

// Queue is based on https://github.com/golang-ds/queue
type Queue[T any] struct {
	data []T
}

// NewQueue constructs and returns an empty slice-queue.
func NewQueue[T any]() *Queue[T] {
	return new(Queue[T])
}

// Enqueue adds an element to the end of the queue.
func (q *Queue[T]) Enqueue(data T) {
	q.data = append(q.data, data)
}

// Dequeue removes and returns the front element of the queue. It returns false if the queue was empty.
func (q *Queue[T]) Dequeue() (val T, ok bool) {
	if q.IsEmpty() {
		return
	}

	val = q.data[0]
	q.data = q.data[1:]

	return val, true
}

// First returns the front element of the queue. It returns false if the queue was empty.
func (q *Queue[T]) First() (val T, ok bool) {
	if q.IsEmpty() {
		return
	}

	return q.data[0], true
}

// Size returns the number of the elements in the queue.
func (q *Queue[T]) Size() int {
	return len(q.data)
}

// Clear empties the queue.
func (q *Queue[T]) Clear() {
	q.data = nil
}

// IsEmpty returns true if the queue is empty.
func (q *Queue[T]) IsEmpty() bool {
	return q.Size() == 0
}
