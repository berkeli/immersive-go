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

var TestDbData = []Image{
	{
		Title:   "A cute kitten",
		AltText: "A kitten looking mischievous",
		Url:     "https://placekitten.com/200/300",
	},
	{
		Title:   "A cute puppy",
		AltText: "A puppy looking mischievous",
		Url:     "https://placedog.net/200/300",
	},
}

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
		err := decoder.Decode(&got)

		require.NoError(t, err)

		require.ElementsMatch(t, got, TestDbData)
	})

	t.Run("POST /images.json", func(t *testing.T) {

		title, altText, url := "title", "alt_text", "url"

		want := Image{
			Title:   title,
			AltText: altText,
			Url:     url,
		}

		b, err := json.Marshal(want)

		require.NoError(t, err)

		request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(b))
		require.NoError(t, err)

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

		require.NoError(t, err)

		newImage := []byte(`{"Title":"New test image","AltText":"alt_text","Url":"test"}`)

		request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(newImage))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		imagesAfter, err := FetchImages(s.conn)

		require.NoError(t, err)

		require.Equal(t, len(imagesBefore)+1, len(imagesAfter))

		img := Image{
			Title:   "New test image",
			AltText: "alt_text",
			Url:     "test",
		}

		require.Contains(t, imagesAfter, img)

	})

	t.Cleanup(closer)
}

func TestFetchImages(t *testing.T) {
	teardownSuite, closer := setupSuite(t)
	defer teardownSuite(t)

	conn := ConnectToDB()

	images, err := FetchImages(conn)

	require.NoError(t, err)

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
