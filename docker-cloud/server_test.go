package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer(t *testing.T) {
	s := Server{}

	t.Run("IndexHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		s.IndexHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("IndexHandler returned wrong status code: got %v want %v",
				w.Code, http.StatusOK)
		}

		expected := "Hello, world!"
		if w.Body.String() != expected {
			t.Errorf("IndexHandler returned unexpected body: got %v want %v",
				w.Body.String(), expected)
		}
	})

	t.Run("PingHandler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ping", nil)
		w := httptest.NewRecorder()

		s.PingHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("PingHandler returned wrong status code: got %v want %v",
				w.Code, http.StatusOK)
		}

		expected := "Pong!"
		if w.Body.String() != expected {
			t.Errorf("PingHandler returned unexpected body: got %v want %v",
				w.Body.String(), expected)
		}
	})
}
