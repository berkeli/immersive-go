package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	t.Run("GET /", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(IndexHandler)

		handler.ServeHTTP(rr, request)

		AssertStatusOK(t, rr)
	})

	t.Run("POST /", func(t *testing.T) {
		bodyHTML := "<em>Hello Test</em>"
		b := strings.NewReader(bodyHTML)
		request, _ := http.NewRequest(http.MethodPost, "/", b)
		request.Header.Add("Content-Type", "text/html")

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(IndexHandler)

		handler.ServeHTTP(rr, request)

		AssertStatusOK(t, rr)

		AssertBodyContains(t, rr, bodyHTML)
	})

	t.Run("GET /200", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/200", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(Handle200)

		handler.ServeHTTP(rr, request)

		AssertStatusOK(t, rr)
	})

	t.Run("GET /500", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/500", nil)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(Handle500)

		handler.ServeHTTP(rr, request)

		AssertStatusInternalServerError(t, rr)
	})

	t.Run("GET /authenticated", func(t *testing.T) {
		t.Run("no authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(HandleAuthenticated("test", "test"))

			handler.ServeHTTP(rr, request)

			AssertStatusUnauthorized(t, rr)
		})

		t.Run("invalid authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			request.Header.Add("Authorization", "Basic invalid")
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(HandleAuthenticated("test", "test"))

			handler.ServeHTTP(rr, request)

			AssertStatusUnauthorized(t, rr)
		})

		t.Run("valid authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			request.Header.Add("Authorization", "Basic dGVzdHVzZXI6c29tZXN0cm9uZ1BXRA==")
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(HandleAuthenticated("testuser", "somestrongPWD"))

			handler.ServeHTTP(rr, request)

			AssertStatusOK(t, rr)
			AssertBodyContains(t, rr, "Hello, testuser")
		})
	})

	t.Run("GET /limited", func(t *testing.T) {
		t.Run("test over the limit", func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, "/limited", nil)
			handler := http.HandlerFunc(HandleRateLimit)
			var responses []httptest.ResponseRecorder
			for i := 0; i < 100; i++ {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, request)
				responses = append(responses, *rr)
			}

			statusOK, statusTooManyRequests := 0, 0
			for i := 0; i < 100; i++ {
				if responses[i].Code == http.StatusOK {
					statusOK++
				} else if responses[i].Code == http.StatusTooManyRequests {
					statusTooManyRequests++
				}
			}

			if statusOK != 70 {
				t.Errorf("got %d want %d", statusOK, 10)
			}

			if statusTooManyRequests != 30 {
				t.Errorf("got %d want %d", statusTooManyRequests, 30)
			}
		})
	})
}

func AssertBodyContains(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !strings.Contains(rr.Body.String(), want) {
		t.Errorf("got %s, want body to include %s", rr.Body.String(), want)
	}
}

func AssertStatusOK(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
}

func AssertStatusInternalServerError(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func AssertStatusUnauthorized(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d want %d", rr.Code, http.StatusUnauthorized)
	}
}

func AssertStatusTooManyRequests(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("got %d want %d", rr.Code, http.StatusTooManyRequests)
	}
}
