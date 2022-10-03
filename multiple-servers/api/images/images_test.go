package images

import (
	"context"
	. "multiple-servers/api/test_utils"
	. "multiple-servers/api/types"

	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAll(t *testing.T) {
	conn, teardownSuite := SetupSuite(t)
	defer teardownSuite(t)

	t.Run("returns all images", func(t *testing.T) {
		images, err := GetAll(conn)
		require.NoError(t, err)
		require.Equal(t, 2, len(images))
		require.ElementsMatch(t, TestDbData, images)
	})

	t.Run("no error when there are no images", func(t *testing.T) {
		_, err := conn.Exec(context.Background(), "DELETE from images")
		require.NoError(t, err)
		images, err := GetAll(conn)
		require.NoError(t, err)
		require.Equal(t, 0, len(images))
	})

}

func TestInserOne(t *testing.T) {
	conn, teardownSuite := SetupSuite(t)
	defer teardownSuite(t)

	t.Run("inserts an image", func(t *testing.T) {
		newImage := Image{
			Title:   "A cute kitten 2",
			AltText: "A kitten looking mischievous 2",
			Url:     "https://placekitten.com/200/300",
		}
		err := InsertOne(conn, newImage)
		require.NoError(t, err)

		images, err := GetAll(conn)
		require.NoError(t, err)

		require.Equal(t, 3, len(images))
		require.Contains(t, images, newImage)
		require.ElementsMatch(t, append(TestDbData, newImage), images)
	})
}
