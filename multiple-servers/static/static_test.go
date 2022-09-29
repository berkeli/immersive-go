package static

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("Return err without path", func(t *testing.T) {

		err := Run(Config{
			Port: 8081,
		})

		require.Error(t, err)
	})

	t.Run("Return err with invalid path", func(t *testing.T) {

		err := Run(Config{
			Port: 8081,
			Path: "asdasd",
		})

		require.Error(t, err)
	})
}

func TestValidatePath(t *testing.T) {
	t.Run("no path", func(t *testing.T) {
		_, err := validatePath("")
		require.Error(t, err)
		require.ErrorContains(t, err, "Path cannot be empty")
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := validatePath("asdsad")
		require.Error(t, err)
		require.ErrorContains(t, err, "Invalid path provided")
	})

	t.Run("valid path", func(t *testing.T) {
		path, err := validatePath("../assets")
		require.NoError(t, err)
		require.Equal(t, "../assets", path)
	})
}
