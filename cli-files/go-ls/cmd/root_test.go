package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var TestTable = map[string]struct {
	Args  []string
	Want  *string
	Error string
}{
	"No path provided": {
		Args:  []string{},
		Want:  StringToPointer("root.go\nroot_test.go\n"),
		Error: "",
	},
	"Relative Path provided": {
		Args:  []string{"../assets"},
		Want:  StringToPointer("dew.txt\nfoo bar\nfor_you.txt\nrain.txt\n"),
		Error: "",
	},
	"File name provided": {
		Args:  []string{"../assets/dew.txt"},
		Want:  StringToPointer("../assets/dew.txt\n"),
		Error: "",
	},
	"Invalid path provided": {
		Args:  []string{"../assets/invalid"},
		Want:  nil,
		Error: ErrInvalidPath,
	},
	"output with -m flag": {
		Args:  []string{"-m", "../assets"},
		Want:  StringToPointer("dew.txt, foo bar, for_you.txt, rain.txt\n"),
		Error: "",
	},
	"output with -m flag and multiple folders": {
		Args:  []string{"-m", "../assets", "../cmd"},
		Want:  StringToPointer("../assets:\ndew.txt, foo bar, for_you.txt, rain.txt\n\n../cmd:\nroot.go, root_test.go\n"),
		Error: "",
	},
}

func TestGolsCmdFunc(t *testing.T) {

	for name, tt := range TestTable {
		t.Run(name, func(t *testing.T) {
			b := bytes.NewBufferString("")
			// Run the command
			golsCmd := &cobra.Command{
				Use:  "go-ls",
				RunE: GolsCmdFunc,
			}

			GoLsCmdFlags(golsCmd)

			golsCmd.SetOut(b)
			golsCmd.SetArgs(tt.Args)
			err := golsCmd.Execute()

			if tt.Error != "" {
				require.ErrorContains(t, err, tt.Error)
			} else {
				require.NoError(t, err)
			}

			// Check the output
			if tt.Want != nil {
				require.Equal(t, *tt.Want, b.String())
			}

		})
	}
}

func StringToPointer(s string) *string {
	return &s
}
