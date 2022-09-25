package routes

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

func TestMain(t *testing.T) {
	t.Run("GET /", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler := IndexHandler(io.ReadAll)
		handler(rr, request)

		AssertStatusOK(t, rr)
	})

	t.Run("POST /", func(t *testing.T) {
		t.Run("should return status 200", func(t *testing.T) {
			bodyHTML := "<em>Hello Test</em>"
			b := strings.NewReader(bodyHTML)
			request, _ := http.NewRequest(http.MethodPost, "/", b)
			request.Header.Add("Content-Type", "text/html")

			rr := httptest.NewRecorder()

			handler := IndexHandler(io.ReadAll)
			handler(rr, request)

			AssertStatusOK(t, rr)

			want := `<!DOCTYPE html>
<html>
&lt;em&gt;Hello Test&lt;/em&gt;`

			AssertBodyEquals(t, rr, want)
		})

		t.Run("should return status 500", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(""))
			request.Header.Add("Content-Type", "text/html")

			rr := httptest.NewRecorder()

			mockReader := func(r io.Reader) ([]byte, error) {
				return nil, errors.New("error")
			}

			handler := IndexHandler(mockReader)

			handler(rr, request)

			AssertStatusInternalServerError(t, rr)
			AssertBodyEquals(t, rr, "Error reading request body")
		})

	})

	t.Run("GET /200", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/200", nil)
		rr := httptest.NewRecorder()

		Handle200(rr, request)

		AssertStatusOK(t, rr)
	})

	t.Run("GET /500", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/500", nil)
		rr := httptest.NewRecorder()

		Handle500(rr, request)

		AssertStatusInternalServerError(t, rr)
	})

	t.Run("GET /authenticated", func(t *testing.T) {
		t.Run("no authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			rr := httptest.NewRecorder()

			handler := HandleAuthenticated("test", "test")

			handler(rr, request)

			AssertStatusUnauthorized(t, rr)
		})

		t.Run("invalid authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			request.SetBasicAuth("test", "invalid")
			rr := httptest.NewRecorder()

			handler := HandleAuthenticated("test", "test")

			handler(rr, request)

			AssertStatusUnauthorized(t, rr)
		})

		t.Run("valid authorisation provided", func(t *testing.T) {

			username, password := "testuser", "somestrongPWD!"

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)

			request.SetBasicAuth(username, password)
			rr := httptest.NewRecorder()

			handler := HandleAuthenticated(username, password)

			handler(rr, request)

			AssertStatusOK(t, rr)
			AssertBodyEquals(t, rr, "Hello, testuser")
		})
	})

	t.Run("GET /limited", func(t *testing.T) {
		t.Run("test over the limit", func(t *testing.T) {
			limiter := rate.NewLimiter(100, 30)
			r := 100 //number of requests to the server
			request, _ := http.NewRequest(http.MethodGet, "/limited", nil)
			handler := HandleRateLimit(limiter)
			var responses []httptest.ResponseRecorder
			for i := 0; i < r; i++ {
				rr := httptest.NewRecorder()
				handler(rr, request)
				responses = append(responses, *rr)
			}

			statusOK, statusTooManyRequests := 0, 0
			for _, response := range responses {
				if response.Code == http.StatusOK {
					statusOK++
				} else if response.Code == http.StatusTooManyRequests {
					statusTooManyRequests++
				} else {
					t.Errorf("got %d want %d or %d", response.Code, http.StatusOK, http.StatusTooManyRequests)
				}
			}

			if statusOK < 30 {
				t.Errorf("got %d wanted at least %d successful requests", statusOK, 30)
			}

			if statusTooManyRequests < 1 {
				t.Errorf("got %d wanted at least %d request to fail", statusTooManyRequests, 1)
			}

			if statusTooManyRequests+statusOK != r {
				t.Errorf("made %d requests, but only received %d responses", r, statusOK+statusTooManyRequests)
			}
		})
	})

	t.Run("Get /limited when over the limit", func(t *testing.T) {
		limiter := rate.NewLimiter(0, 0)
		request, _ := http.NewRequest(http.MethodGet, "/limited", nil)
		rr := httptest.NewRecorder()
		handler := HandleRateLimit(limiter)
		handler(rr, request)

		AssertStatusTooManyRequests(t, rr)
	})
}

func AssertBodyEquals(t *testing.T, rr *httptest.ResponseRecorder, want string) {
	t.Helper()
	if rr.Body.String() != want {
		t.Errorf("got %s, want body to be %s", rr.Body.String(), want)
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
