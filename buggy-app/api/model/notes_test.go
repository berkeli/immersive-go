package model

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/require"
)

func TestTags(t *testing.T) {

	tests := map[string]struct {
		text     string
		expected []string
	}{
		"no tags": {
			text:     "This is an example",
			expected: []string{},
		},
		"one tag": {
			text:     "This is an example #tag1",
			expected: []string{"tag1"},
		},
		"two tags": {
			text:     "This is an example #tag1 #tag2",
			expected: []string{"tag1", "tag2"},
		},
		"two tags with spaces": {
			text:     "This is an example #tag1    #tag2    ",
			expected: []string{"tag1", "tag2"},
		},
		"tag at the start": {
			text:     "#tag1 This is an example",
			expected: []string{"tag1"},
		},
		"tag with comma": {
			text:     "This is an example #tag1,#tag2",
			expected: []string{"tag1", "tag2"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tags := extractTags(test.text)

			if !reflect.DeepEqual(test.expected, tags) {
				t.Fatalf("expected %v, got %v", test.expected, tags)
			}
		})
	}
}

func TestGetNotesForOwner(t *testing.T) {

	currTime := time.Now()

	tests := map[string]struct {
		owner       string
		rows        *pgxmock.Rows
		expected    Notes
		expectedErr string
	}{
		"no notes": {
			owner: "test",
			rows:  pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}),
		},
		"one note": {
			owner:    "test",
			rows:     pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}).AddRow("id1", "test", "content1", currTime, currTime),
			expected: Notes{{Id: "id1", Owner: "test", Content: "content1", Created: currTime, Modified: currTime, Tags: []string{}}},
		},
		"note with tags": {
			owner:    "test",
			rows:     pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}).AddRow("id1", "test", "content1 #tag1", currTime, currTime),
			expected: Notes{{Id: "id1", Owner: "test", Content: "content1 #tag1", Created: currTime, Modified: currTime, Tags: []string{"tag1"}}},
		},
		"no owner": {
			owner:       "",
			expectedErr: "owner not supplied",
		},
		"error": {
			owner:       "test",
			rows:        pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}).RowError(0, fmt.Errorf("some error")),
			expectedErr: "some error",
		},
	}

	mock, err := pgxmock.NewPool(pgxmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if test.rows != nil {
				mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE owner = (.+)$").
					WithArgs(test.owner).
					WillReturnRows(test.rows)
			}

			notes, err := GetNotesForOwner(ctx, mock, test.owner)

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
			}

			require.ElementsMatch(t, test.expected, notes, "expected %v, got %v", test.expected, notes)
		})
	}
}

func TestGetNoteById(t *testing.T) {
	currTime := time.Now()
	tests := map[string]struct {
		id          string
		rows        *pgxmock.Rows
		expected    Note
		expectedErr string
	}{
		"no note": {
			id:   "id1",
			rows: pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}),
		},
		"valid id": {
			id:   "id1",
			rows: pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}).AddRow("id1", "test", "content1", currTime, currTime),
			expected: Note{
				Id:       "id1",
				Owner:    "test",
				Content:  "content1",
				Tags:     []string{},
				Created:  currTime,
				Modified: currTime,
			},
		},
		"valid id with tags": {
			id:   "id1",
			rows: pgxmock.NewRows([]string{"id", "owner", "content", "created", "modified"}).AddRow("id1", "test", "content1 #tag1", currTime, currTime),
			expected: Note{
				Id:       "id1",
				Owner:    "test",
				Content:  "content1 #tag1",
				Tags:     []string{"tag1"},
				Created:  currTime,
				Modified: currTime,
			},
		},
		"no id": {
			expectedErr: "id not supplied",
		},
	}

	mock, err := pgxmock.NewPool(pgxmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if test.rows != nil {
				mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE id = (.+)$").
					WithArgs(test.id).
					WillReturnRows(test.rows)
			}

			note, err := GetNoteById(ctx, mock, test.id)

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
			}

			require.Equal(t, test.expected, note, "expected %v, got %v", test.expected, note)
		})
	}
}
