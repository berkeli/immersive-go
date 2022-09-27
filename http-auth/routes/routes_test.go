package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

type MockReader struct{}

func (r MockReader) Read(p []byte) (int, error) {
	return 0, errors.New("some error")
}

func TestMain(t *testing.T) {

	c := Controllers{
		Username: "testUN",
		Password: "testPWD",
		Limiter:  rate.NewLimiter(100, 30),
	}

	t.Run("GET /", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		c.IndexHandler(rr, request)

		AssertStatusOK(t, rr)
	})

	t.Run("POST /", func(t *testing.T) {
		t.Run("should return status 200", func(t *testing.T) {
			bodyHTML := "<em>Hello Test</em>"
			b := strings.NewReader(bodyHTML)
			request, _ := http.NewRequest(http.MethodPost, "/", b)
			request.Header.Add("Content-Type", "text/html")

			rr := httptest.NewRecorder()

			c.IndexHandler(rr, request)

			AssertStatusOK(t, rr)

			want := `<!DOCTYPE html>
<html>
&lt;em&gt;Hello Test&lt;/em&gt;`

			AssertBodyEquals(t, rr, want)
		})

		t.Run("should return status 500", func(t *testing.T) {

			m := MockReader{}

			request, _ := http.NewRequest(http.MethodPost, "/", m)
			request.Header.Add("Content-Type", "text/html")

			rr := httptest.NewRecorder()

			c.IndexHandler(rr, request)

			AssertStatusInternalServerError(t, rr)
			AssertBodyEquals(t, rr, "Error reading request body")
		})

	})

	t.Run("GET /200", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/200", nil)
		rr := httptest.NewRecorder()

		c.Handle200(rr, request)

		AssertStatusOK(t, rr)
	})

	t.Run("GET /500", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/500", nil)
		rr := httptest.NewRecorder()

		c.Handle500(rr, request)

		AssertStatusInternalServerError(t, rr)
	})

	t.Run("GET /authenticated", func(t *testing.T) {
		t.Run("no authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			rr := httptest.NewRecorder()

			c.HandleAuthenticated(rr, request)

			AssertStatusUnauthorized(t, rr)
		})

		t.Run("invalid authorisation provided", func(t *testing.T) {

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)
			request.SetBasicAuth("test", "invalid")
			rr := httptest.NewRecorder()

			c.HandleAuthenticated(rr, request)

			AssertStatusUnauthorized(t, rr)
		})

		t.Run("valid authorisation provided", func(t *testing.T) {

			username, password := "testUN", "testPWD"

			request, _ := http.NewRequest(http.MethodGet, "/authenticated", nil)

			request.SetBasicAuth(username, password)
			rr := httptest.NewRecorder()

			c.HandleAuthenticated(rr, request)

			AssertStatusOK(t, rr)
			AssertBodyEquals(t, rr, "Hello, testUN")
		})
	})

	t.Run("GET /limited", func(t *testing.T) {
		t.Run("test over the limit", func(t *testing.T) {
			r := 100 //number of requests to the server
			request, _ := http.NewRequest(http.MethodGet, "/limited", nil)
			var responses []httptest.ResponseRecorder
			for i := 0; i < r; i++ {
				rr := httptest.NewRecorder()
				c.HandleRateLimit(rr, request)
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
		c = Controllers{
			Limiter: rate.NewLimiter(0, 0),
		}
		request, _ := http.NewRequest(http.MethodGet, "/limited", nil)
		rr := httptest.NewRecorder()
		c.HandleRateLimit(rr, request)

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
