package images

import (
	"context"
	. "multiple-servers/api/types"

	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/require"
)

var TEST_DB_URL = "postgres://berkeli:postgres@localhost:5432/go_server_test_db"

var TestDbData = []Image{
	{
		Title:   "A cute kitten",
		AltText: "A kitten looking mischievous",
		Url:     "https://placekitten.com/200/300",
	},
	{
		Title:   "A cute puppy",
		AltText: "A puppy looking mischievous",
		Url:     "https://placedog.net/200/300",
	},
}

func setupSuite(tb testing.TB) func(tb testing.TB) {

	conn, err := pgx.Connect(context.Background(), TEST_DB_URL)
	require.NoError(tb, err)

	_, err = conn.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS images (id serial, title text, alt_text text, url text)")
	require.NoError(tb, err)

	_, err = conn.Exec(context.Background(), "DELETE from images")
	require.NoError(tb, err)

	for _, image := range TestDbData {
		_, err = conn.Exec(context.Background(), "INSERT INTO images (title, alt_text, url) VALUES ($1, $2, $3)", image.Title, image.AltText, image.Url)
		require.NoError(tb, err)
	}

	require.NoError(tb, err)

	return func(tb testing.TB) {
		// teardown the database after testing
		_, err := conn.Exec(context.Background(), "DELETE from images")

		require.NoError(tb, err)
		conn.Close(context.Background())
	}
}

func TestGetAll(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	conn, err := pgx.Connect(context.Background(), TEST_DB_URL)
	defer conn.Close(context.Background())
	require.NoError(t, err)

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
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	conn, err := pgx.Connect(context.Background(), TEST_DB_URL)
	defer conn.Close(context.Background())
	require.NoError(t, err)

	t.Run("inserts an image", func(t *testing.T) {
		newImage := Image{
			Title:   "A cute kitten 2",
			AltText: "A kitten looking mischievous 2",
			Url:     "https://placekitten.com/200/300",
		}
		_, err := InsertOne(conn, newImage)
		require.NoError(t, err)

		images, err := GetAll(conn)
		require.NoError(t, err)

		require.Equal(t, 3, len(images))
		require.Contains(t, images, newImage)
		require.ElementsMatch(t, append(TestDbData, newImage), images)
	})
}
