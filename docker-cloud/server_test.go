package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {

	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "could not connect to Docker")

	resource, err := pool.Run("docker-cloud", "latest", []string{})
	require.NoError(t, err, "could not start container")

	t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "failed to remove container")
	})

	t.Run("Index should return hello world", func(t *testing.T) {

		var resp *http.Response

		err = pool.Retry(func() error {
			resp, err = http.Get(fmt.Sprint("http://localhost:", resource.GetPort("80/tcp"), "/"))
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

		require.Equal(t, "Hello, world!", string(body))
	})

	t.Run("Ping should return pong", func(t *testing.T) {

		var resp *http.Response

		err = pool.Retry(func() error {
			resp, err = http.Get(fmt.Sprint("http://localhost:", resource.GetPort("80/tcp"), "/ping"))
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

		require.Equal(t, "Pong!", string(body))
	})
}
