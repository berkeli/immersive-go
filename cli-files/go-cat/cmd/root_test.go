package cmd

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestRootCmd(t *testing.T) {
	t.Run("No path provided", func(t *testing.T) {

		cmd := rootCmd
		err := cmd.Execute()

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("path to file dew.txt", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := rootCmd
		cmd.SetArgs([]string{"../assets/dew.txt"})
		err := cmd.Execute()

		w.Close()

		if err != nil {
			t.Error("Did not expect error, got", err)
		}

		want := `“A World of Dew” by Kobayashi Issa

A world of dew,
And within every dewdrop
A world of struggle.
`

		got, err := ioutil.ReadAll(r)

		if string(got) != want {
			t.Errorf("Expected %q, got %q", want, string(got))
		}

	})
}
