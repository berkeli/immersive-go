package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type Test struct {
	image Image
	err   error
}

func TestImages(t *testing.T) {

	t.Run("GetAll", func(t *testing.T) {
		conn, teardownSuite := setupSuite(t)
		defer teardownSuite(t)

		i := Images{
			conn: conn,
		}

		images, err := i.GetAll()

		require.NoError(t, err)
		require.Len(t, images, 2)
		require.ElementsMatch(t, images, TestDbData)
	})

	t.Run("InsertOne", func(t *testing.T) {
		conn, teardownSuite := setupSuite(t)
		defer teardownSuite(t)

		i := Images{
			conn: conn,
		}

		testTable := map[string]Test{
			"Invalid URL": {
				image: Image{
					Title:   "title",
					AltText: "alt_text",
					Url:     "url",
				},
				err: fmt.Errorf("Provided URL is not valid: url"),
			},
			"Duplicate URL": {
				image: Image{
					Title:   "title",
					AltText: "alt_text",
					Url:     "https://placedog.net/200/300",
				},
				err: fmt.Errorf("Image with the same URL already exists in Database: https://placedog.net/200/300"),
			},
			"Valid Image": {
				image: Image{
					Title:   "title",
					AltText: "alt_text",
					Url:     "https://placedog.net/200/500",
				},
				err: nil,
			},
		}

		for name, test := range testTable {
			t.Run(name, func(t *testing.T) {

				got := i.InsertOne(test.image)

				require.Equal(t, test.err, got)
			})
		}
	})
}
