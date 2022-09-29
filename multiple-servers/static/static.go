package static

import (
	"fmt"
	"net/http"
	"os"
)

type Config struct {
	Path string `json:"path"`
	Port int    `json:"port"`
}

func Run(c Config) error {

	path, err := validatePath(c.Path)

	if err != nil {
		return err
	}

	handler := http.FileServer(http.Dir(path))

	http.Handle("/", handler)

	err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)

	if err != nil {
		return err
	}

	return nil
}

func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("Path cannot be empty")
	}

	file, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", "Invalid path provided", err)
	}

	if !file.IsDir() {
		return path, fmt.Errorf("%s: %w", "Path is not a directory", err)
	}

	return path, nil
}
