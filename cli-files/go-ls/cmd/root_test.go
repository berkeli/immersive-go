package cmd

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGolsCmdFunc(t *testing.T) {
	t.Run("No path provided", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := golsCmd
		cmd.Execute()

		w.Close()
		want := "root.go\nroot_test.go\n"

		got, _ := ioutil.ReadAll(r)

		AssertStringsEqual(t, string(got), want)
	})

	t.Run("Relative Path provided", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := golsCmd
		cmd.SetArgs([]string{"../assets"})
		cmd.Execute()

		w.Close()
		want := "dew.txt\nfor_you.txt\nrain.txt\n"

		got, _ := ioutil.ReadAll(r)

		AssertStringsEqual(t, string(got), want)
	})

	t.Run("File name provided", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := golsCmd
		cmd.SetArgs([]string{"../assets/dew.txt"})
		cmd.Execute()

		w.Close()
		want := "../assets/dew.txt\n"

		got, _ := ioutil.ReadAll(r)

		AssertStringsEqual(t, string(got), want)
	})

	t.Run("Invalid path provided", func(t *testing.T) {

		cmd := golsCmd
		cmd.SetArgs([]string{"../assets/invalid"})
		err := cmd.Execute()

		if err == nil {
			t.Errorf("expected an error but didn't get one")
		}
	})

	t.Run("output with -m flag", func(t *testing.T) {
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := golsCmd
		cmd.SetArgs([]string{"-m", "../assets"})
		cmd.Execute()

		w.Close()
		want := "dew.txt, for_you.txt, rain.txt\n"

		got, _ := ioutil.ReadAll(r)

		AssertStringsEqual(t, string(got), want)
	})
}

func AssertStringsEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
