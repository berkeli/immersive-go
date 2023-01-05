package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/gographics/imagick.v2/imagick"
)

func TestGrayscaleMockError(t *testing.T) {
	c := &Converter{
		cmd: func(args []string) (*imagick.ImageCommandResult, error) {
			return nil, errors.New("not implemented")
		},
	}

	err := c.Grayscale("input.jpg", "output.jpg")
	require.Error(t, err)
}

func TestGrayscaleMockCall(t *testing.T) {
	var args []string
	expected := []string{"convert", "input.jpg", "-set", "colorspace", "Gray", "output.jpg"}
	c := &Converter{
		cmd: func(a []string) (*imagick.ImageCommandResult, error) {
			args = a
			return &imagick.ImageCommandResult{
				Info: nil,
				Meta: "",
			}, nil
		},
	}

	err := c.Grayscale("input.jpg", "output.jpg")
	require.NoError(t, err)
	require.Exactly(t, expected, args)
}
