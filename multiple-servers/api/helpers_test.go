package api

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
		require.Equal(t, err, errors.New("Indent cannot be negative: -1. Please provide a positive number. Default is 1"))
	})

	t.Run("Invalid indent", func(t *testing.T) {
		_, err := ValidateIndent("a")
		require.Error(t, err)
		require.Equal(t, err, errors.New("Unable to parse indent: a. Please provide a positive number. Default is 1"))
	})
}

func TestSerializeIndented(t *testing.T) {
	t.Run("indent is 2", func(t *testing.T) {
		data := struct {
			Name string
		}{
			Name: "John",
		}
		s, err := serializeIndented(data, 2)
		want := `{
  "Name": "John"
}`

		require.NoError(t, err)
		require.Equal(t, want, string(s))
	})

	t.Run("indent is 0", func(t *testing.T) {
		data := struct {
			Name string
		}{
			Name: "John",
		}
		s, err := serializeIndented(data, 0)
		want := `{"Name":"John"}`

		require.NoError(t, err)
		require.Equal(t, want, string(s))
	})
}
