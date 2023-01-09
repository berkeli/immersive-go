package server

import (
	"testing"
)

func Test_Storage(t *testing.T) {
	t.Run("TestMapStorage", func(t *testing.T) {
		s := NewMapStorage()

		if s.HasData() {
			t.Errorf("Expected HasData() to be false on empty storage")
		}

		val := []byte("bar")

		s.Set("foo", val)
		if got, ok := s.Get("foo"); !ok || string(got) != string(val) {
			t.Errorf("Expected bar, got %s", got)
		}

		if !s.HasData() {
			t.Errorf("Expected HasData() to be true")
		}
	})
}
