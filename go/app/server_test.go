package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"os"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/mock/gomock"
	"encoding/json"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	//"log"
	"context"
	
)

func TestParseAddItemRequest(t *testing.T) {
	t.Parallel()

	imageBytes, err := os.ReadFile("../images/default.jpg")
	if err != nil {
		t.Fatalf("failed to read image file: %v", err)
	}

	type wants struct {
		req *AddItemRequest
		err bool
	}

	// STEP 6-1: define test cases
	cases := map[string]struct {
		args map[string]string
		wants
	}{
		"ok: valid request": {
			args: map[string]string{
				"name":     "jaket_test",
				"category": "fashion_test",
				"image":    "../images/default.jpg",
			},
			wants: wants{
				req: &AddItemRequest{
					Name:     "jaket_test",
					Category: "fashion_test",
					Image:    imageBytes,
				},
				err: false,
			},
		},
		"ng: empty request": {
			args: map[string]string{},
			wants: wants{
				req: nil,
				err: true,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// prepare request body
			values := url.Values{}
			for k, v := range tt.args {
				values.Set(k, v)
			}

			// prepare HTTP request
			req, err := http.NewRequest("POST", "http://localhost:9000/items", strings.NewReader(values.Encode()))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// execute test target
			got, err := parseAddItemRequest(req)

			// confirm the result
			if err != nil {
				if !tt.err {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if diff := cmp.Diff(tt.wants.req, got); diff != "" {
				t.Errorf("unexpected request (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelloHandler(t *testing.T) {
	t.Parallel()

	type wants struct {
		code int               // desired HTTP status code
		body map[string]string // desired body
	}
	want := wants{
		code: http.StatusOK,
	 	body: map[string]string{"message": "Hello, world!"},
	}

	// set up test
	req := httptest.NewRequest("GET", "/hello", nil)
	res := httptest.NewRecorder()

	h := &Handlers{}
	h.Hello(res, req)

	// STEP 6-2: confirm the status code
	if res.Code != want.code {
		t.Errorf("expected status code %d, got %d", want.code, res.Code)
	}

	// STEP 6-2: confirm response body
	var gotBody map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &gotBody); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if diff := cmp.Diff(want.body, gotBody); diff != "" {
		t.Errorf("unexpected response body (-want +got):\n%s", diff)
	}
}

func TestAddItem(t *testing.T) {
	t.Parallel()

	type wants struct {
		code int
	}
	cases := map[string]struct {
		args     map[string]string
		injector func(m *MockItemRepository)
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
                m.EXPECT().
                    GetCategoryID(gomock.Any(), "phone").
                    Return(1, nil)

                m.EXPECT().
                    Insert(gomock.Any(), gomock.Any()).
                    DoAndReturn(func(ctx context.Context, item *Item) error {
                        // check the inserted item
                        if item.Name != "used iPhone 16e" || item.Category != "phone" {
                            return errors.New("invalid item")
                        }
                        return nil
                    })
                m.EXPECT().List(gomock.Any()).Return([]Item{
                    {Name: "used iPhone 16e", Category: "phone"},
                }, nil)
				// succeeded to insert
			},
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				m.EXPECT().
                    GetCategoryID(gomock.Any(), "phone").
                    Return(1, nil)

                m.EXPECT().
                    Insert(gomock.Any(), gomock.Any()).
                    Return(errors.New("failed to insert"))
				// failed to insert
			},
			wants: wants{
				code: http.StatusInternalServerError,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockIR := NewMockItemRepository(ctrl)
			tt.injector(mockIR)
			h := &Handlers{imgDirPath: "../images", itemRepo: mockIR}

			values := url.Values{}
			for k, v := range tt.args {
				values.Set(k, v)
			}
			req := httptest.NewRequest("POST", "/items", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()
			h.AddItem(rr, req)

			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}

			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}
		})
	}
}

// STEP 6-4: uncomment this test
func TestAddItemE2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	db, closers, err := setupDB(t)
	if err != nil {
		t.Fatalf("failed to set up database: %v", err)
	}
	t.Cleanup(func() {
		for _, c := range closers {
			c()
		}
	})

	type wants struct {
		code int
	}
	cases := map[string]struct {
		args map[string]string
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "",
				"category": "phone",
			},
			wants: wants{
				code: http.StatusBadRequest,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			h := &Handlers{imgDirPath: "../images", itemRepo: &itemRepository{db: db}}

			values := url.Values{}
			for k, v := range tt.args {
				values.Set(k, v)
			}
			req := httptest.NewRequest("POST", "/items", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()
			h.AddItem(rr, req)

			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}
			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}

			// STEP 6-4: check inserted data
			var items []Item
			var category_id int

			rows, err := db.Query(`
				SELECT i.id, i.name, i.category_id, i.image_name 
				FROM items i
				JOIN categories c ON i.category_id = c.id
			`)
			
			if err != nil {
				t.Fatalf("failed to query items: %v", err)
			}
			defer rows.Close()

			for rows.Next() {
				var item Item
				if err := rows.Scan(&item.ID, &item.Name, &category_id, &item.ImageName); err != nil {
					t.Fatalf("failed to scan item: %v", err)
				}
				items = append(items, item)
			}

			if err := rows.Err(); err != nil {
				t.Fatalf("error during row iteration: %v", err)
			}

			if len(items) != 1 {
				t.Errorf("expected 1 item, got %d", len(items))
			}
		})
	}
}

func setupDB(t *testing.T) (db *sql.DB, closers []func(), e error) {
	t.Helper()

	defer func() {
		if e != nil {
			for _, c := range closers {
				c()
			}
		}
	}()

	// create a temporary file for e2e testing
	f, err := os.CreateTemp(".", "*.sqlite3")
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		f.Close()
		os.Remove(f.Name())
	})

	// set up tables
	db, err = sql.Open("sqlite3", f.Name())
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		db.Close()
	})

	// TODO: replace it with real SQL statements.
	// cmd := `CREATE TABLE IF NOT EXISTS items (
	// 	id INTEGER PRIMARY KEY AUTOINCREMENT,
	// 	name VARCHAR(255),
	// 	category VARCHAR(255)
	// )`

	// create categories table
	cmd := `CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`
	_, err = db.Exec(cmd)
	if err != nil {
		return nil, nil, err
	}

	// insert test category
	cmd = `INSERT INTO categories (name) VALUES ('phone')`
	_, err = db.Exec(cmd)
	if err != nil {
		return nil, nil, err
	}
	// create items table
	cmd = `CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		image_name TEXT NOT NULL,
		FOREIGN KEY (category_id) REFERENCES categories(id)
	)`
	_, err = db.Exec(cmd)
	if err != nil {
		return nil, nil, err
	}
	return db, closers, nil
}