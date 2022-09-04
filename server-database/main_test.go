package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestImage(t *testing.T) {
	image := Image{"title", "altText", "url"}

	AssertStrings(t, image.Title, "title")

	AssertStrings(t, image.AltText, "altText")

	AssertStrings(t, image.Url, "url")

}

func TestLoadENV(t *testing.T) {
	closer := envSetter(map[string]string{
		"EXECUTION_ENVIRONMENT": "test",
	})
	LoadENV()
	AssertStrings(t, os.Getenv("EXECUTION_ENVIRONMENT"), "test")
	t.Cleanup(closer)
}

func TestConnectToDB(t *testing.T) {
	closer := envSetter(map[string]string{
		"EXECUTION_ENVIRONMENT": "test",
		"DB_URL":                "postgresql://postgres:postgres@localhost:5432/go_server_test_db",
	})

	LoadENV()
	ConnectToDB()
	defer conn.Close(context.Background())

	t.Cleanup(closer)
}

func TestMain(t *testing.T) {

	closer := envSetter(map[string]string{
		"EXECUTION_ENVIRONMENT": "test",
		"DB_URL":                "postgresql://postgres:postgres@localhost:5432/go_server_test_db",
	})

	LoadENV()
	ConnectToDB()

	t.Run("Get /", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()

		IndexHandler(response, request)

		got := response.Body.String()
		want := "Hello World"

		AssertStrings(t, got, want)
	})

	t.Run("Get /images.json", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/images.json", nil)
		response := httptest.NewRecorder()

		ImagesHandler(response, request)

		var got []Image

		decoder := json.NewDecoder(response.Body)
		decoder.Decode(&got)

		if len(got) == 0 {
			t.Errorf("Expected GET /images.json to have images, received %v", len(got))
		}
	})

	t.Run("POST /images.json", func(t *testing.T) {

		want := Image{"title", "alt_text", "url"}

		b, _ := json.Marshal(want)

		request, _ := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(b))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		ImagesHandler(response, request)

		var got Image

		decoder := json.NewDecoder(response.Body)
		decoder.Decode(&got)

		AssertImages(t, got, want)

	})

	t.Cleanup(closer)
}

func AssertStrings(t *testing.T, got, want string) {
	if got != want {
		t.Errorf("Expected %s, received %s", want, got)
	}
}

func AssertImages(t *testing.T, got, want Image) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %v, received %v", want, got)
	}
}

func envSetter(envs map[string]string) (closer func()) {
	originalEnvs := map[string]string{}

	for name, value := range envs {
		if originalValue, ok := os.LookupEnv(name); ok {
			originalEnvs[name] = originalValue
		}
		_ = os.Setenv(name, value)
	}

	return func() {
		for name := range envs {
			origValue, has := originalEnvs[name]
			if has {
				_ = os.Setenv(name, origValue)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	}
}
