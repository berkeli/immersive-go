package images

import (
	"context"
	"log"
	. "multiple-servers/api/types"

	"github.com/jackc/pgx/v4"
)

func GetAll(conn *pgx.Conn) ([]Image, error) {
	rows, err := conn.Query(context.Background(), "SELECT title, alt_text, url FROM images")
	if err != nil {
		log.Printf("Unable to fetch images: %s", err.Error())
	}

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url)
		if err != nil {
			log.Printf("Unable to scan row: %v", err)
		}
		images = append(images, image)
	}

	return images, err
}

func Insert(conn *pgx.Conn, image Image) (Image, error) {
	rows, err := conn.Query(context.Background(), "SELECT title, alt_text, url FROM images")
	if err != nil {
		log.Printf("Unable to fetch images: %s", err.Error())
	}

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url)
		if err != nil {
			log.Printf("Unable to scan row: %v", err)
		}
		images = append(images, image)
	}

	return images[0], err
}
