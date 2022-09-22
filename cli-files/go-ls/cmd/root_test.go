package cmd

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

var TestTable = map[string]struct {
	Args  []string
	Want  *string
	Error string
}{
	"No path provided": {
		Args:  []string{},
		Want:  StringToPointer("root.go\nroot_test.go"),
		Error: "",
	},
	"Relative Path provided": {
		Args:  []string{"../assets"},
		Want:  StringToPointer("dew.txt\nfoo bar\nfor_you.txt\nrain.txt"),
		Error: "",
	},
	"File name provided": {
		Args:  []string{"../assets/dew.txt"},
		Want:  StringToPointer("../assets/dew.txt"),
		Error: "",
	},
	"Invalid path provided": {
		Args:  []string{"../assets/invalid"},
		Want:  nil,
		Error: ErrInvalidPath,
	},
	"output with -m flag": {
		Args:  []string{"-m", "../assets"},
		Want:  StringToPointer("dew.txt, foo bar, for_you.txt, rain.txt"),
		Error: "",
	},
	"output with -m flag and multiple folders": {
		Args:  []string{"-m", "../assets", "../cmd"},
		Want:  StringToPointer("../assets:\ndew.txt, foo bar, for_you.txt, rain.txt\n../cmd:\nroot.go, root_test.go"),
		Error: "",
	},
}

func TestGolsCmdFunc(t *testing.T) {

	for name, tt := range TestTable {
		t.Run(name, func(t *testing.T) {
			b := bytes.NewBufferString("")

			// Run the command
			golsCmd.SetOut(b)
			golsCmd.SetArgs(tt.Args)
			err := golsCmd.Execute()

			if tt.Error != "" {
				require.ErrorContains(t, err, tt.Error)
			} else {
				require.NoError(t, err)
			}

			got, err := ioutil.ReadAll(b)

			if err != nil {
				t.Fatal(err)
			}

			// Check the output
			if tt.Want != nil {
				require.Equal(t, *tt.Want, string(got))
			}

		})
	}
}

func StringToPointer(s string) *string {
	return &s
}
