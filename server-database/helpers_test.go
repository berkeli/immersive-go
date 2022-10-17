package main

import (
	"errors"
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
	}{
		"Valid jpg image": {
			url:      "https://www.pakainfo.com/wp-content/uploads/2021/09/image-url-for-testing.jpg",
			expected: nil,
		},
		"Valid png image": {
			url:      "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png",
			expected: errors.New("Unable to decode image: image: unknown format"),
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
			_, _, err := ValidateImage(test.url)
			require.Equal(t, test.expected, err)
		})
	}

}
