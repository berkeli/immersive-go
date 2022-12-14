package types

import (
	"fmt"
	"testing"
)

func Test_Command(t *testing.T) {

	allowedClusters := map[string]struct{}{
		"cluster-a": {},
		"cluster-b": {},
	}

	tests := map[string]struct {
		cmd     Command
		wantErr error
	}{
		"should return an error if schedule is empty": {
			cmd: Command{
				Description: "hello world",
				Command:     "echo hello world",
				MaxRetries:  3,
			},
			wantErr: fmt.Errorf("schedule is required"),
		},
		"should return an error if command is empty": {
			cmd: Command{
				Description: "hello world",
				Schedule:    "*/1 * * * *",
				MaxRetries:  3,
			},
			wantErr: fmt.Errorf("command is required"),
		},
		"should return an error if max retries is greater than 3": {
			cmd: Command{
				Description: "hello world",
				Schedule:    "*/1 * * * *",
				Command:     "echo hello world",
				MaxRetries:  4,
			},
		},
		"should return an error if cluster is not allowed": {
			cmd: Command{
				Clusters:    []string{"cluster-c"},
				Description: "hello world",
				Schedule:    "*/1 * * * *",
				Command:     "echo hello world",
				MaxRetries:  3,
			},
			wantErr: fmt.Errorf("cluster 'cluster-c' is not recognized"),
		},
		"valid Command should not return error": {
			cmd: Command{
				Clusters:    []string{"cluster-a"},
				Description: "hello world",
				Schedule:    "*/1 * * * *",
				Command:     "echo hello world",
				MaxRetries:  3,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.cmd.Validate(allowedClusters, 3)

			if err != nil && tc.wantErr != nil {
				if err.Error() != tc.wantErr.Error() {
					t.Errorf("got %s, want %s", err.Error(), tc.wantErr.Error())
				}
			}
		})
	}
}
