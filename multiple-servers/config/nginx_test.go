package nginx_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

func TestNginx(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "could not connect to Docker")

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "multi-aio",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	require.NoError(t, err, "could not start container")

	t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "failed to remove container")
	})

	t.Run("test server is running", func(t *testing.T) {
		var resp *http.Response

		err = pool.Retry(func() error {
			resp, err = http.Get("http://localhost:" + resource.GetPort("8080/tcp"))
			if err != nil {
				t.Log("container not ready, waiting...")
				return err
			}
			return nil
		})

		require.NoError(t, err, "could not connect to container")

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected status code to be 200")

		resp.Body.Close()
	})

	t.Run("test api is running", func(t *testing.T) {
		var resp *http.Response

		err = pool.Retry(func() error {
			resp, err = http.Get(fmt.Sprintf("http://localhost:%s/api/", resource.GetPort("8080/tcp")))
			if err != nil {
				t.Log("container not ready, waiting...")
				return err
			}
			return nil
		})

		require.NoError(t, err, "could not connect to container")

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected status code to be 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "could not read response body")

		require.Equal(t, "Welcome to the API", string(body), "expected body to be 'Welcome to the API'")

		resp.Body.Close()
	})
}
