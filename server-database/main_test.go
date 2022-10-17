package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImage(t *testing.T) {

	title, altText, url := "title", "title alt_text", "url"

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

	conn, teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	s := &Server{
		images: &Images{
			conn: conn,
		},
	}

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

	t.Run("Get /images.json with indent", func(t *testing.T) {

		i := 2

		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/images.json?indent=%d", i), nil)
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		require.Equal(t, http.StatusOK, response.Code)

		got := response.Body.String()

		prefixCurly := fmt.Sprintf("%s%s", strings.Repeat(" ", i), "{")
		prefixString := fmt.Sprintf("%s%s", strings.Repeat(" ", i*2), "\"")

		prefixCurlyCount := strings.Count(got, prefixCurly)
		prefixStringCount := strings.Count(got, prefixString)

		//when there are 2 items in the databse, there should be 2 curly brace matches and 6 string matches (3 per item)
		require.GreaterOrEqual(t, prefixCurlyCount, 2)
		require.GreaterOrEqual(t, prefixStringCount, 6)
	})

	t.Run("Get /images.json with invalid indent", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/images.json?indent=-2", nil)
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		require.Equal(t, response.Result().StatusCode, http.StatusBadRequest)

		want := "Indent cannot be negative: -2"

		require.Equal(t, want, response.Body.String())
	})

	t.Run("POST /images.json", func(t *testing.T) {

		title, altText, url := "title", "title alt_text", "https://placedog.net/200/400"

		want := Image{
			Title:   title,
			AltText: altText,
			Url:     url,
			Width:   200,
			Height:  400,
		}

		b := []byte(`{"Title": "title", "AltText": "title alt_text", "Url": "https://placedog.net/200/400"}`)

		request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(b))
		require.NoError(t, err)

		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		require.Equal(t, response.Result().StatusCode, http.StatusCreated)

		var got Image

		decoder := json.NewDecoder(response.Body)
		decoder.Decode(&got)

		require.Equal(t, want, got)

	})

	t.Run("POST /images.json invalid format", func(t *testing.T) {

		b := []byte(`{"Title": "title", "AltText": "title alt_text", "Url": "url`)

		request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(b))
		require.NoError(t, err)

		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		require.Equal(t, response.Result().StatusCode, http.StatusBadRequest)

	})

	t.Run("POST /images.json and fetch from DB", func(t *testing.T) {

		imagesBefore, err := s.images.GetAll()

		require.NoError(t, err)

		newImage := []byte(`{"Title":"Test","AltText":"test alt_text","Url":"https://placedog.net/200/500"}`)

		request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer(newImage))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		s.ImagesHandler(response, request)

		imagesAfter, err := s.images.GetAll()

		require.NoError(t, err)

		require.Equal(t, len(imagesBefore)+1, len(imagesAfter))

		img := Image{
			Title:   "Test",
			AltText: "test alt_text",
			Url:     "https://placedog.net/200/500",
			Width:   200,
			Height:  500,
		}

		require.Contains(t, imagesAfter, img)

	})

}
