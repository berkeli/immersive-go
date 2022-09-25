package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const DB_URL = "postgresql://postgres:postgres@localhost:5432/go_server_test_db"

func setupSuite(tb testing.TB) (func(tb testing.TB), func()) {

	// setup the database for testing

	closer := envSetter(map[string]string{
		"DB_URL": DB_URL,
	})

	conn := ConnectToDB()

	_, err := conn.Exec(context.Background(), "DELETE from images")

	if err != nil {
		tb.Fatalf("Setup Error: Unable to delete from images: %s", err.Error())
	}

	_, err = conn.Exec(context.Background(), `INSERT INTO images (title, url, alt_text) 
		VALUES ('A cute kitten', 'https://placekitten.com/200/300', 'A kitten looking mischievous'), 
		('A cute puppy', 'https://placedog.net/200/300', 'A puppy looking mischievous')`,
	)

	if err != nil {
		tb.Fatalf("Setup Error: Unable to insert into images: %s", err.Error())
	}

	return func(tb testing.TB) {
		// teardown the database after testing
		_, err := conn.Exec(context.Background(), "DELETE from images")

		if err != nil {
			tb.Fatalf("Teardown Error: Unable to delete from images: %s", err.Error())
		}
	}, closer
}

func TestImage(t *testing.T) {

	title, altText, url := "title", "alt_text", "url"

	image := Image{
		Title:   title,
		AltText: altText,
		Url:     url,
	}

	require.Equal(t, title, image.Title)
	require.Equal(t, altText, image.AltText)
	require.Equal(t, url, image.Url)
}

func TestMain(t *testing.T) {

	teardownSuite, closer := setupSuite(t)
	defer teardownSuite(t)

	s := &Server{conn: ConnectToDB()}

	t.Run("Get /", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()

		s.IndexHandler(response, request)

		got := response.Body.String()
		want := "Hello World"

		require.Equal(t, want, got)
	})

	t.Run("Get /images.json", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/images.json", nil)
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		var got []Image

		decoder := json.NewDecoder(response.Body)
		decoder.Decode(&got)

		var TestDbData = []Image{
			{"A cute kitten", "A kitten looking mischievous", "https://placekitten.com/200/300"},
			{"A cute puppy", "A puppy looking mischievous", "https://placedog.net/200/300"},
		}

		require.ElementsMatch(t, got, TestDbData)
	})

	t.Run("POST /images.json", func(t *testing.T) {

		want := Image{"title", "alt_text", "url"}

		b, _ := json.Marshal(want)

		request, _ := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(b))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		var got Image

		decoder := json.NewDecoder(response.Body)
		decoder.Decode(&got)

		require.Equal(t, want, got)

	})

	t.Run("POST /images.json and fetch from DB", func(t *testing.T) {

		imagesBefore, err := FetchImages(s.conn)

		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}

		newImage := Image{"New test image", "test", "test"}

		b, err := json.Marshal(newImage)
		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}

		request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(b))
		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		imagesAfter, err := FetchImages(s.conn)

		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}

		require.Equal(t, len(imagesBefore)+1, len(imagesAfter))

		require.Contains(t, imagesAfter, newImage)

	})

	t.Cleanup(closer)
}

func TestFetchImages(t *testing.T) {
	teardownSuite, closer := setupSuite(t)
	defer teardownSuite(t)

	conn := ConnectToDB()

	images, err := FetchImages(conn)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	var TestDbData = []Image{
		{"A cute kitten", "A kitten looking mischievous", "https://placekitten.com/200/300"},
		{"A cute puppy", "A puppy looking mischievous", "https://placedog.net/200/300"},
	}

	require.ElementsMatch(t, images, TestDbData)

	t.Cleanup(closer)
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
