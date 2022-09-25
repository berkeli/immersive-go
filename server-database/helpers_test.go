package main

import (
	"context"
	"testing"
)

func TestConnectToDB(t *testing.T) {
	closer := envSetter(map[string]string{
		"DB_URL": DB_URL,
	})

	conn := ConnectToDB()
	defer conn.Close(context.Background())

	t.Cleanup(closer)
}

func TestValidateIndent(t *testing.T) {
	t.Run("Valid indent", func(t *testing.T) {
		i, err := ValidateIndent("2")
		if err != nil {
			t.Errorf("Expected indent to be valid, received %v", err)
		}
		if i != 2 {
			t.Errorf("Expected indent to be 2, received %v", i)
		}
	})

	t.Run("Negative indent", func(t *testing.T) {
		_, err := ValidateIndent("-1")
		if err == nil {
			t.Errorf("Expected indent to be invalid, received %v", err)
		}

		if err.Error() != "Indent cannot be negative: -1" {
			t.Errorf("Expected: Indent cannot be negative: -1, received %v", err.Error())
		}
	})

	t.Run("Invalid indent", func(t *testing.T) {
		_, err := ValidateIndent("a")
		if err == nil {
			t.Errorf("Expected indent to be invalid, received %v", err)
		}

		if err.Error() != "Unable to parse indent: a" {
			t.Errorf("Expected: Unable to parse indent: a, received %v", err.Error())
		}
	})
}
