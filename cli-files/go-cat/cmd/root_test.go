package cmd

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

var TestTable = []struct {
	Name           string
	Args           []string
	ExpectedOutput string
	ExpectedError  error
	AssertOutput   bool
}{
	{
		Name:           "No path provided",
		Args:           []string{},
		ExpectedOutput: "",
		ExpectedError:  ErrNoPath,
		AssertOutput:   false,
	},
	{
		Name: "Path to dew.txt",
		Args: []string{"../assets/dew.txt"},
		ExpectedOutput: `“A World of Dew” by Kobayashi Issa

A world of dew,
And within every dewdrop
A world of struggle.
`,
		ExpectedError: nil,
		AssertOutput:  true,
	},
}

func TestRootCmd(t *testing.T) {

	for _, tt := range TestTable {
		t.Run(tt.Name, func(t *testing.T) {
			b := bytes.NewBufferString("")

			rootCmd.SetOut(b)

			rootCmd.SetArgs(tt.Args)
			err := rootCmd.Execute()
			require.ErrorIs(t, err, tt.ExpectedError)

			if tt.AssertOutput {
				out, err := ioutil.ReadAll(b)
				if err != nil {
					t.Fatal(err)
				}

				require.Equal(t, string(out), tt.ExpectedOutput)
			}
		})
	}

}
