package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadPasswd(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		t.Setenv("POSTGRES_PASSWORD", "password")

		passwd, err := ReadPasswd()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		require.Equal(t, passwd, "password")
	})

	t.Run("from file", func(t *testing.T) {
		file, err := os.CreateTemp("", "password")
		file.Write([]byte("password123"))
		file.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Setenv("POSTGRES_PASSWORD_FILE", file.Name())

		passwd, err := ReadPasswd()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		require.Equal(t, passwd, "password123")

		t.Cleanup(func() {
			os.Remove(file.Name())
		})
	})

	t.Run("from non-existent file", func(t *testing.T) {
		t.Setenv("POSTGRES_PASSWORD_FILE", "non-existent-file")

		_, err := ReadPasswd()
		require.Error(t, err)
	})
}
