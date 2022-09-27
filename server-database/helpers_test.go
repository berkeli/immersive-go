package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectToDB(t *testing.T) {
	conn := ConnectToDB(TEST_DB_URL)
	conn.Close(context.Background())
}

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
