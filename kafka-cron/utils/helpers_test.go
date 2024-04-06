package utils

import (
	"os"
	"testing"

	"github.com/berkeli/kafka-cron/types"
	"github.com/stretchr/testify/require"
)

func Test_WithTopicPrefix(t *testing.T) {
	tests := map[string]struct {
		prefix  string
		cluster string
		want    string
	}{
		"empty prefix": {
			prefix:  "",
			cluster: "cluster-a",
			want:    "jobs-cluster-a",
		},
		"non-empty prefix": {
			prefix:  "prefix",
			cluster: "cluster-a",
			want:    "prefix-cluster-a",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			os.Setenv("TOPIC_PREFIX", tc.prefix)
			defer os.Unsetenv("TOPIC_PREFIX")

			cluster := tc.cluster
			want := tc.want

			got := WithTopicPrefix()(cluster)

			if got != want {
				t.Errorf("got %s, want %s", got, want)
			}
		})
	}
}

func Test_ReadConfig(t *testing.T) {
	sample := `
max_allowed_retries: 3
clusters:
  - name: "cluster-a"
crons:
  - description: 'Echo Hello world every day at 00:00 AM'
    schedule: "0 0 * * *"
    command: "echo 'Hello, world!'"
    max_retries: 3
    clusters: 
      - "cluster-a"
      - "cluster-b"
  - description: 'Echo Hello world every minute'
    schedule: "* * * * *"
    command: echo 'Hello, world!'
    clusters: 
      - "cluster-b"`

	want := types.ConfigFile{
		MaxAllowedRetries: 3,
		Clusters: []types.Cluster{
			{
				Name: "cluster-a",
			},
		},
		Crons: []types.Command{
			{
				Clusters:    []string{"cluster-a", "cluster-b"},
				Description: "Echo Hello world every day at 00:00 AM",
				Command:     "echo 'Hello, world!'",
				MaxRetries:  3,
				Schedule:    "0 0 * * *",
			},
			{
				Clusters:    []string{"cluster-b"},
				Description: "Echo Hello world every minute",
				Command:     "echo 'Hello, world!'",
				MaxRetries:  0,
				Schedule:    "* * * * *",
			},
		},
	}

	f, err := os.CreateTemp(".", "config.yaml")

	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(f.Name())

	_, err = f.WriteString(sample)

	if err != nil {
		t.Fatal(err)
	}

	got, err := ReadConfig(f.Name())

	if err != nil {
		t.Fatal(err)
	}

	require.ElementsMatch(t, want.Crons, got.Crons)
	require.ElementsMatch(t, want.Clusters, got.Clusters)
	require.Equal(t, want.MaxAllowedRetries, got.MaxAllowedRetries)
}
