package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

var TestTable = map[string]struct {
	endpoint string
	expected string
}{
	"Index should return hello world": {
		endpoint: "/",
		expected: "Hello, world!",
	},
	"Ping should return pong": {
		endpoint: "/ping",
		expected: "Pong!",
	},
}

func TestServer(t *testing.T) {

	tag := os.Getenv("DOCKER_TAG")

	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "could not connect to Docker")

	resource, err := pool.Run("docker-cloud", tag, []string{})
	require.NoError(t, err, "could not start container")

	// t.Cleanup(func() {
	// 	require.NoError(t, pool.Purge(resource), "failed to remove container")
	// })

	for name, tt := range TestTable {
		t.Run(name, func(t *testing.T) {

			var resp *http.Response

			err = pool.Retry(func() error {
				resp, err = http.Get(fmt.Sprintf("http://localhost:%s%s", resource.GetPort("80/tcp"), tt.endpoint))
				if err != nil {
					t.Log("container not ready, waiting...")
					return err
				}
				return nil
			})

			require.NoError(t, err)

			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expected, string(body))
		})
	}

}
