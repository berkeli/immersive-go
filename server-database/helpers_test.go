package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateIndent(t *testing.T) {
	t.Run("Valid indent", func(t *testing.T) {
		i, err := ValidateIndent("2")
		require.NoError(t, err)
		require.Equal(t, 2, i)
	})

	t.Run("Negative indent", func(t *testing.T) {
		_, err := ValidateIndent("-1")
		require.Error(t, err)
		require.Equal(t, err, errors.New("Indent cannot be negative: -1"))
	})

	t.Run("Invalid indent", func(t *testing.T) {
		_, err := ValidateIndent("a")
		require.Error(t, err)
		require.Equal(t, err, errors.New("Unable to parse indent: a"))
	})
}

func TestValidateAltText(t *testing.T) {
	t.Run("Valid alt text", func(t *testing.T) {
		err := ValidateAltText("This is a title", "This is an alt text")
		require.NoError(t, err)
	})

	t.Run("Invalid alt text", func(t *testing.T) {
		err := ValidateAltText("This is a title", "Alt about Something else entirely")
		require.Error(t, err)
		require.Equal(t, errors.New("Alt text doesn't seem to be relevant to the title"), err)
	})

	t.Run("short alt text", func(t *testing.T) {
		err := ValidateAltText("This is a title", "hello")
		require.Error(t, err)
		require.Equal(t, err, errors.New("Alt text must contain at least as many words as the title"))
	})
}

func TestValidateImage(t *testing.T) {

	testTable := map[string]struct {
		url      string
		expected error
		width    int
		height   int
	}{
		"Valid jpg image": {
			url:      "https://placekitten.com/200/300",
			expected: nil,
			width:    200,
			height:   300,
		},
		"Valid png image": {
			url:      "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png",
			expected: nil,
			width:    544,
			height:   184,
		},
		"Invalid image": {
			url:      "https://www.google.com/",
			expected: errors.New("Unable to decode image: image: unknown format"),
		},
		"Invalid url": {
			url:      "https://www.google.com/invalid",
			expected: errors.New("Unable to fetch image: https://www.google.com/invalid"),
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			w, h, err := ValidateImage(test.url)
			require.Equal(t, test.expected, err)
			require.Equal(t, test.width, w)
			require.Equal(t, test.height, h)
		})
	}

	t.Run("test against local server", func(t *testing.T) {
		fs := http.FileServer(http.Dir("test_assets"))

		svr := httptest.NewServer(fs)

		defer svr.Close()

		testTable := map[string]struct {
			path   string
			err    error
			width  int
			height int
		}{
			"Valid jpg image": {
				path:   "test.jpg",
				err:    nil,
				width:  100,
				height: 20,
			},
			"Valid png image": {
				path:   "test.png",
				err:    nil,
				width:  100,
				height: 20,
			},
			"Valid gif image": {
				path:   "test.gif",
				err:    nil,
				width:  100,
				height: 20,
			},
			"Invalid image": {
				path: "test.txt",
				err:  errors.New("Unable to decode image: image: unknown format"),
			},
			"Invalid URL": {
				path: "test",
				err:  fmt.Errorf("Unable to fetch image: %s/test", svr.URL),
			},
		}

		for name, test := range testTable {
			t.Run(name, func(t *testing.T) {
				w, h, err := ValidateImage(svr.URL + "/" + test.path)
				require.Equal(t, test.err, err)
				require.Equal(t, test.width, w)
				require.Equal(t, test.height, h)
			})
		}
	})

}

func Test_decodeImage(t *testing.T) {
	testTable := map[string]struct {
		path   string
		err    error
		width  int
		height int
	}{
		"Valid jpg image": {
			path:   "test_assets/test.jpg",
			err:    nil,
			width:  100,
			height: 20,
		},
		"Valid png image": {
			path:   "test_assets/test.png",
			err:    nil,
			width:  100,
			height: 20,
		},
		"Valid gif image": {
			path:   "test_assets/test.gif",
			err:    nil,
			width:  100,
			height: 20,
		},
		"Invalid image": {
			path: "test_assets/test.txt",
			err:  errors.New("image: unknown format"),
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			file, err := os.Open(test.path)

			require.NoError(t, err)

			w, h, err := decodeImage(file)
			require.Equal(t, test.err, err)
			require.Equal(t, test.width, w)
			require.Equal(t, test.height, h)
		})
	}
}
