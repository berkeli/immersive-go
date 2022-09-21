package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var TestTable = map[string]struct {
	Args           []string
	AssertOutput   bool
	ExpectedOutput string
	ExpectedError  error
}{
	"No path provided": {
		Args:           []string{},
		AssertOutput:   false,
		ExpectedOutput: "",
		ExpectedError:  ErrNoPath,
	},
	"Path to dew.txt": {
		Args:         []string{"../assets/dew.txt"},
		AssertOutput: true,
		ExpectedOutput: `“A World of Dew” by Kobayashi Issa

A world of dew,
And within every dewdrop
A world of struggle.`,
		ExpectedError: nil,
	},
}

func TestRootCmd(t *testing.T) {

	for name, tt := range TestTable {
		t.Run(name, func(t *testing.T) {
			b := bytes.NewBufferString("")

			rootCmd.SetOut(b)

			rootCmd.SetArgs(tt.Args)
			err := rootCmd.Execute()
			require.ErrorIs(t, err, tt.ExpectedError)

			if tt.AssertOutput {
				require.Equal(t, tt.ExpectedOutput, b.String())
			}
		})
	}

}
