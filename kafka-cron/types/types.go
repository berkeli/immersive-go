package types

import (
	"fmt"

	"github.com/robfig/cron/v3"
)

type ConfigFile struct {
	MaxAllowedRetries int       `json:"max_allowed_retries" yaml:"max_allowed_retries"`
	Clusters          []Cluster `json:"clusters" yaml:"clusters"`
	Crons             []Command `json:"crons" yaml:"crons"`
}

type Cluster struct {
	Name        string `json:"name" yaml:"name"`
	Partitions  int    `json:"partitions" yaml:"partitions"`
	Replication int    `json:"replication" yaml:"replication"`
}

type Command struct {
	Description string   `json:"description" yaml:"description"`
	Schedule    string   `json:"schedule" yaml:"schedule"`
	Command     string   `json:"command" yaml:"command"`
	MaxRetries  int      `json:"max_retries" yaml:"max_retries"`
	Clusters    []string `json:"clusters" yaml:"clusters"`
}

func (c *Command) Validate(allowedClusters map[string]struct{}, maxRetries int) error {
	if c.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}

	if _, err := cron.ParseStandard(c.Schedule); err != nil {
		return fmt.Errorf("schedule is invalid: %v", err)
	}

	if c.Command == "" {
		return fmt.Errorf("command is required")
	}

	if c.MaxRetries > maxRetries {
		return fmt.Errorf("max_retries is greater than max allowed")
	}

	for _, cluster := range c.Clusters {
		if _, ok := allowedClusters[cluster]; !ok {
			return fmt.Errorf("cluster '%s' is not recognised", cluster)
		}
	}
	return nil
}
