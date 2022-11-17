package model

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Note struct {
	Id       string    `json:"id"`
	Owner    string    `json:"owner"`
	Content  string    `json:"content"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Tags     []string  `json:"tags"`
}

type Notes []Note

type dbConn interface {
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

func GetNotesForOwner(ctx context.Context, conn dbConn, owner string, page, per_page int) (Notes, int, error) {
	if owner == "" {
		return nil, 0, errors.New("model: owner not supplied")
	}

	queryRows, err := conn.Query(ctx, "SELECT id, owner, content, created, modified FROM public.note WHERE owner = $1 ORDER BY created DESC LIMIT $2 OFFSET $3", owner, per_page, page*per_page)
	if err != nil {
		return nil, 0, fmt.Errorf("model: could not query notes: %w", err)
	}
	defer queryRows.Close()

	notes := []Note{}
	for queryRows.Next() {
		note := Note{}
		err = queryRows.Scan(&note.Id, &note.Owner, &note.Content, &note.Created, &note.Modified)
		if err != nil {
			return nil, 0, fmt.Errorf("model: query scan failed: %w", err)
		}
		note.Tags = extractTags(note.Content)
		notes = append(notes, note)
	}

	if queryRows.Err() != nil {
		return nil, 0, fmt.Errorf("model: query read failed: %w", queryRows.Err())
	}

	queryRow := conn.QueryRow(ctx, "SELECT COUNT(*) FROM public.note WHERE owner = $1", owner)
	count := 0
	err = queryRow.Scan(&count)

	if err != nil {
		return nil, 0, fmt.Errorf("model: could not query notes: %w", err)
	}

	return notes, count, nil
}

func GetNoteById(ctx context.Context, conn dbConn, id string) (Note, error) {
	var note Note
	if id == "" {
		return note, errors.New("model: id not supplied")
	}

	row := conn.QueryRow(ctx, "SELECT id, owner, content, created, modified FROM public.note WHERE id = $1", id)

	err := row.Scan(&note.Id, &note.Owner, &note.Content, &note.Created, &note.Modified)
	if err != nil {
		return note, fmt.Errorf("model: query scan failed: %w", err)
	}
	note.Tags = extractTags(note.Content)
	return note, nil
}

// Extract tags from the note. We're looking for #something. There could be
// multiple tags, so we FindAll.
func extractTags(input string) []string {
	re := regexp.MustCompile(`#([a-zA-Z0-9(_)]{1,})`)
	matches := re.FindAllStringSubmatch(input, -1)
	tags := make([]string, 0, len(matches))
	for _, f := range matches {
		tags = append(tags, strings.TrimSpace(f[1]))
	}
	return tags
}