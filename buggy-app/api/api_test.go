package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CodeYourFuture/immersive-go-course/buggy-app/api/model"
	"github.com/CodeYourFuture/immersive-go-course/buggy-app/auth"
	"github.com/CodeYourFuture/immersive-go-course/buggy-app/util"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v2"
)

var defaultConfig Config = Config{
	Port:           8090,
	Log:            log.Default(),
	AuthServiceUrl: "auth:8080",
}

type envelopeNote struct {
	Note model.Note `json:"note"`
}

func assertJSON(actual []byte, data interface{}, t *testing.T) {
	expected, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when marshaling expected json data", err)
	}

	if !bytes.Equal(expected, actual) {
		t.Errorf("the expected json: %s is different from actual %s", expected, actual)
	}
}

func TestRun(t *testing.T) {
	as := New(defaultConfig)

	var runErr error
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = as.Run(ctx)
	}()

	<-time.After(1000 * time.Millisecond)
	cancel()

	wg.Wait()
	if runErr != http.ErrServerClosed {
		t.Fatal(runErr)
	}
}

func TestSimpleRequest(t *testing.T) {
	as := New(defaultConfig)

	var runErr error
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr = as.Run(ctx)
	}()

	<-time.After(1000 * time.Millisecond)

	resp, err := http.Get("http://localhost:8090/1/my/notes.json")
	if err != nil {
		cancel()
		wg.Wait()
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		cancel()
		wg.Wait()
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	cancel()
	wg.Wait()
	if runErr != http.ErrServerClosed {
		t.Fatal(runErr)
	}
}

func TestMyNotesAuthFail(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateDeny,
	})

	req, err := http.NewRequest("GET", "/1/my/notes.json", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}
	res := httptest.NewRecorder()
	handler := http.HandlerFunc(as.handleMyNotes)
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestMyNotesAuthFailWithAuth(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateDeny,
	})

	req, err := http.NewRequest("GET", "/1/my/notes.json", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Basic ZXhhbXBsZTpleGFtcGxl")
	res := httptest.NewRecorder()
	handler := http.HandlerFunc(as.handleMyNotes)
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestMyNotesAuthFailMalformedAuth(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateDeny,
	})

	req, err := http.NewRequest("GET", "/1/my/notes.json", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Basic nope")
	res := httptest.NewRecorder()
	handler := as.Handler()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestMyNotesAuthPass(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateAllow,
	})

	rows := mock.NewRows([]string{"id", "owner", "content"})

	mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE owner = (.+)?$").WillReturnRows(rows)
	mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE owner = (.+)?$").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).
			AddRow(15))

	req, err := http.NewRequest("GET", "/1/my/notes.json", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Basic ZXhhbXBsZTpleGFtcGxl")

	res := httptest.NewRecorder()
	handler := as.Handler()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestMyNotesOneNone(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateAllow,
	})

	id, password := "abc123", "password"
	noteId, content, created, modified := "xyz789", "Note content", time.Now(), time.Now()

	rows := mock.NewRows([]string{"id", "owner", "content", "created", "modified"}).
		AddRow(noteId, id, content, created, modified)

	mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE owner = (.+)$").WillReturnRows(rows)
	mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE owner = (.+)?$").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).
			AddRow(15))

	req, err := http.NewRequest("GET", "/1/my/notes.json", strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", util.BasicAuthHeaderValue(id, password))
	res := httptest.NewRecorder()
	handler := as.Handler()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	data := EnvelopeNotes{
		Notes: []model.Note{
			{Id: noteId, Owner: id, Content: content, Created: created, Modified: modified, Tags: []string{}},
		},
		Total:   15,
		Page:    0,
		PerPage: 10,
	}
	assertJSON(res.Body.Bytes(), data, t)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %s", err)
	}
}

func TestMyNoteById(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateAllow,
	})
	id, password := "abc123", "password"
	noteId, content, created, modified := "xyz789", "Note content", time.Now(), time.Now()

	t.Run("valid authorisation", func(t *testing.T) {
		tests := map[string]struct {
			rows         *pgxmock.Rows
			url          string
			expectedCode int
			assertData   bool
			expectedData envelopeNote
		}{
			"note belongs to user": {
				rows:         mock.NewRows([]string{"id", "owner", "content", "created", "modified"}).AddRow(noteId, id, content, created, modified),
				url:          fmt.Sprintf("/1/my/note/%s.json", noteId),
				expectedCode: http.StatusOK,
				assertData:   true,
				expectedData: envelopeNote{Note: model.Note{Id: noteId, Owner: id, Content: content, Created: created, Modified: modified, Tags: []string{}}},
			},
			"note does not belong to user": {
				rows:         mock.NewRows([]string{"id", "owner", "content", "created", "modified"}).AddRow(noteId, "someone-else", content, created, modified),
				url:          fmt.Sprintf("/1/my/note/%s.json", noteId),
				expectedCode: http.StatusBadRequest,
			},
			"note does not exist": {
				rows:         mock.NewRows([]string{"id", "owner", "content", "created", "modified"}).RowError(0, pgx.ErrNoRows),
				url:          fmt.Sprintf("/1/my/note/%s.json", noteId),
				expectedCode: http.StatusNotFound,
			},
			"no note ID provided": {
				url:          "/1/my/note/",
				expectedCode: http.StatusBadRequest,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				if test.rows != nil {
					mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE id = (.+)$").WillReturnRows(test.rows)
				}
				req, err := http.NewRequest("GET", test.url, strings.NewReader(""))
				if err != nil {
					log.Fatal(err)
				}
				req.Header.Add("Authorization", util.BasicAuthHeaderValue(id, password))
				res := httptest.NewRecorder()
				handler := as.Handler()
				handler.ServeHTTP(res, req)

				if res.Code != test.expectedCode {
					t.Fatalf("expected status %d, got %d", test.expectedCode, res.Code)
				}

				if test.assertData {
					assertJSON(res.Body.Bytes(), test.expectedData, t)
				}

				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatalf("unfulfilled expectations: %s", err)
				}
			})
		}
	})
}

func TestMyNoteByIdWithTags(t *testing.T) {
	as := New(defaultConfig)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()
	as.pool = mock
	as.authClient = auth.NewMockClient(&auth.VerifyResult{
		State: auth.StateAllow,
	})

	id, password := "abc123", "password"
	noteId, content, created, modified := "xyz789", "Note content #tag1", time.Now(), time.Now()

	rows := mock.NewRows([]string{"id", "owner", "content", "created", "modified"}).
		AddRow(noteId, id, content, created, modified)

	mock.ExpectQuery("^SELECT (.+) FROM public.note WHERE id = (.+)$").WillReturnRows(rows)

	req, err := http.NewRequest("GET", fmt.Sprintf("/1/my/note/%s.json", noteId), strings.NewReader(""))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", util.BasicAuthHeaderValue(id, password))
	res := httptest.NewRecorder()
	handler := as.Handler()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	data := struct {
		Note model.Note `json:"note"`
	}{Note: model.Note{Id: noteId, Owner: id, Content: content, Created: created, Modified: modified, Tags: []string{"tag1"}}}
	assertJSON(res.Body.Bytes(), data, t)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %s", err)
	}
}
