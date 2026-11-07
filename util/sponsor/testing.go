package sponsor

import "testing"

// SetSubjectForTest sets the subject under lock and restores it when the test ends
func SetSubjectForTest(t testing.TB, subject string) {
	t.Helper()

	mu.Lock()
	old := Subject
	Subject = subject
	mu.Unlock()

	t.Cleanup(func() {
		mu.Lock()
		Subject = old
		mu.Unlock()
	})
}
