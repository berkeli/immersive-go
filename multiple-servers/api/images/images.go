package images

import (
	"context"
	. "multiple-servers/api/types"

	"github.com/jackc/pgx/v4"
)

func GetAll(conn *pgx.Conn) ([]Image, error) {
	rows, err := conn.Query(context.Background(), "SELECT title, alt_text, url FROM images")
	if err != nil {
		return nil, err
	}

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url)
		if err != nil {
			return images, err
		}
		images = append(images, image)
	}

	return images, nil
}

func InsertOne(conn *pgx.Conn, newImage Image) error {
	row := conn.QueryRow(context.Background(), "INSERT INTO images (title, alt_text, url) VALUES ($1, $2, $3) RETURNING title, alt_text, url", newImage.Title, newImage.AltText, newImage.Url)

	var image Image
	err := row.Scan(&image.Title, &image.AltText, &image.Url)
	if err != nil {
		return err
	}

	return nil
}
