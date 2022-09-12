package cmd

import (
	"bytes"
	"io/ioutil"
	"testing"
)

var TestTable = []struct {
	Name  string
	Args  []string
	Want  string
	Error string
}{
	{
		Name:  "No path provided",
		Args:  []string{},
		Want:  "root.go\nroot_test.go\n",
		Error: "",
	},
	{
		Name:  "Relative Path provided",
		Args:  []string{"../assets"},
		Want:  "dew.txt\nfoo bar\nfor_you.txt\nrain.txt\n",
		Error: "",
	},
	{
		Name:  "File name provided",
		Args:  []string{"../assets/dew.txt"},
		Want:  "../assets/dew.txt",
		Error: ErrNotADirectory,
	},
	{
		Name:  "Invalid path provided",
		Args:  []string{"../assets/invalid"},
		Want:  "",
		Error: ErrInvalidPath,
	},
	{
		Name:  "output with -m flag",
		Args:  []string{"-m", "../assets"},
		Want:  "dew.txt, foo bar, for_you.txt, rain.txt\n",
		Error: "",
	},
	{
		Name:  "output with -m flag and multiple folders",
		Args:  []string{"-m", "../assets", "../cmd"},
		Want:  "../assets: \ndew.txt, foo bar, for_you.txt, rain.txt\n\n../cmd: \nroot.go, root_test.go\n",
		Error: "",
	},
}

func TestGolsCmdFunc(t *testing.T) {

	for _, tt := range TestTable {
		t.Run(tt.Name, func(t *testing.T) {
			b := bytes.NewBufferString("")

			// Run the command
			golsCmd.SetOut(b)
			golsCmd.SetArgs(tt.Args)
			err := golsCmd.Execute()

			AssertErrors(t, err, tt.Error)

			got, err := ioutil.ReadAll(b)

			if err != nil {
				t.Fatal(err)
			}

			// Check the output
			if tt.Error == "" {
				AssertStringsEqual(t, string(got), tt.Want)
			}
		})
	}
}

func AssertStringsEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func AssertErrors(t *testing.T, err error, want string) {
	t.Helper()
	if err != nil && err.Error() != want {
		t.Errorf("got %q want %q", err.Error(), want)
	}
}
