package api

import (
	"bytes"
	"context"
	"encoding/json"
	"multiple-servers/api/images"
	. "multiple-servers/api/test_utils"

	. "multiple-servers/api/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	teardown := SetupSuite(t)
	defer teardown(t)

	conn, err := pgx.Connect(context.Background(), TEST_DB_URL)
	require.NoError(t, err)

	s := Server{
		db: conn,
	}

	t.Run("IndexHandler", func(t *testing.T) {
		t.Run("GET", func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, "/", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			s.IndexHandler(rr, request)

			require.Equal(t, http.StatusOK, rr.Code)
			require.Equal(t, "Welcome to the API", rr.Body.String())
		})

		t.Run("POST", func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, "/", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			s.IndexHandler(rr, request)

			require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	})

	var TestTable = map[string]struct {
		requestUrl   string
		method       string
		requestBody  []byte
		expectedCode int
		expectedBody string
		expectedData []Image
	}{
		"GET": {
			requestUrl:   "/images.json",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedBody: `[
 {
  "title": "A cute kitten",
  "alt_text": "A kitten looking mischievous",
  "URL": "https://placekitten.com/200/300"
 },
 {
  "title": "A cute puppy",
  "alt_text": "A puppy looking mischievous",
  "URL": "https://placedog.net/200/300"
 }
]`,
			expectedData: TestDbData,
		},
		"GET with indent": {
			requestUrl:   "/images.json?indent=4",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
			expectedBody: `[
    {
        "title": "A cute kitten",
        "alt_text": "A kitten looking mischievous",
        "URL": "https://placekitten.com/200/300"
    },
    {
        "title": "A cute puppy",
        "alt_text": "A puppy looking mischievous",
        "URL": "https://placedog.net/200/300"
    }
]`,
			expectedData: TestDbData,
		},
		"GET with invalid indent": {
			requestUrl:   "/images.json?indent=-1",
			method:       http.MethodPost,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Indent cannot be negative: -1",
			expectedData: nil,
		},
		"POST with invalid json": {
			requestUrl:   "/images.json",
			method:       http.MethodPost,
			requestBody:  []byte(`{"title": "A cute kitten", "alt_text": "A kitten looking mischievous", "URL": "https://placekitten.com/200/300"`),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Unable to parse json",
			expectedData: nil,
		},
	}

	t.Run("/images.json handler", func(t *testing.T) {
		for name, test := range TestTable {
			t.Run(name, func(t *testing.T) {
				request, err := http.NewRequest(test.method, test.requestUrl, bytes.NewBuffer(test.requestBody))
				require.NoError(t, err)

				rr := httptest.NewRecorder()

				s.ImagesHandler(rr, request)

				require.Equal(t, test.expectedCode, rr.Code)

				if test.expectedData != nil {
					var actual []Image
					err = json.Unmarshal(rr.Body.Bytes(), &actual)
					require.NoError(t, err)

					require.ElementsMatch(t, test.expectedData, actual)
				}
				require.Equal(t, test.expectedBody, rr.Body.String())

			})
		}

		t.Run("return 404 when ther are no images", func(t *testing.T) {
			teardown := SetupSuite(t)
			defer teardown(t)
			_, err := conn.Exec(context.Background(), "DELETE FROM images")
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodGet, "/images.json", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			s.ImagesHandler(rr, request)

			require.Equal(t, http.StatusNotFound, rr.Code)
			require.Equal(t, "No images found", rr.Body.String())
		})

		t.Run("POST with valid json", func(t *testing.T) {
			teardown := SetupSuite(t)
			defer teardown(t)

			newImage := Image{
				Title:   "A cute kitten 2",
				AltText: "A kitten looking mischievous 2",
				Url:     "https://placekitten.com/200/300",
			}

			request, err := http.NewRequest(http.MethodPost, "/images.json", bytes.NewBuffer([]byte(`{"title": "A cute kitten 2", "alt_text": "A kitten looking mischievous 2", "URL": "https://placekitten.com/200/300"}`)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			s.ImagesHandler(rr, request)

			require.Equal(t, http.StatusCreated, rr.Code)

			var actual Image
			err = json.Unmarshal(rr.Body.Bytes(), &actual)
			require.NoError(t, err)

			require.Equal(t, newImage, actual)

			newImages, err := images.GetAll(conn)
			require.NoError(t, err)

			require.ElementsMatch(t, append(TestDbData, newImage), newImages)

		})
	})
}
