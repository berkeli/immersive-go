package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// PG Errors
var (
	InvalidUrl   = "new row for relation \"images\" violates check constraint \"url_is_valid\""
	DuplicateUrl = "duplicate key value violates unique constraint \"images_url_key\""
)

type Images struct {
	conn *pgx.Conn
}

func (i *Images) GetAll() ([]Image, error) {
	rows, err := i.conn.Query(context.Background(), "SELECT title, alt_text, url, width, height FROM images")
	if err != nil {
		log.Printf("Unable to fetch images: %s", err.Error())
		return nil, err
	}

	var images []Image

	for rows.Next() {
		var image Image
		err = rows.Scan(&image.Title, &image.AltText, &image.Url, &image.Width, &image.Height)
		if err != nil {
			log.Printf("Unable to scan row: %v", err)
			return nil, err
		}
		images = append(images, image)
	}

	return images, nil
}

func (i *Images) InsertOne(image Image) error {
	_, err := i.conn.Exec(context.Background(), "INSERT INTO images (title, alt_text, url, width, height) VALUES ($1, $2, $3, $4, $5)", image.Title, image.AltText, image.Url, image.Width, image.Height)
	if err != nil {
		log.Printf("Unable to insert image: %s", err.Error())

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Printf("Error Code: %s", pgErr.Message)
			switch pgErr.Message {
			case InvalidUrl:
				return fmt.Errorf("Provided URL is not valid: %s", image.Url)
			case DuplicateUrl:
				return fmt.Errorf("Image with the same URL already exists in Database: %s", image.Url)
			default:
				return fmt.Errorf("Could not insert image: %s", pgErr.Message)
			}
		}
		return err
	}
	return nil
}
