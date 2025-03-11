package app

import (
	"context"
	//"encoding/json"
	"errors"
	"fmt"
	//"io"
	//"os"
	"path/filepath"
	//"strconv"
	// STEP 5-1: uncomment this line
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
)

//custom error
var (
    errImageNotFound = errors.New("image not found")
    errItemNotFound  = errors.New("item not found")
    errInvalidInput  = errors.New("invalid input")
)

type Item struct {
	ID         int    `db:"id" json:"id"`
	Name       string `db:"name" json:"name"`
	Category   string `db:"category" json:"category"` 
	ImageName  string `db:"image_name" json:"image_name"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error //insert an item
	List(ctx context.Context) ([]Item, error) //get all items
	Get(ctx context.Context, id string) (*Item, error) //get an item by id
	Search(ctx context.Context, keyword string) ([]Item, error) //search items by keyword
	Close() error //close the database connection
	GetCategoryID(ctx context.Context, categoryName string) (int, error) //get category id by name
	GetCategoryName(ctx context.Context, categoryID int) (string, error) //get category name by id
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	//fileName string
	// filePath is the absolute path to the JSON file
	//filePath string
	db *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() (ItemRepository, error) {
	dbPath := filepath.Join("db", "mercari.sqlite3")
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)

    // check if the database is connected
    if err = db.Ping(); err != nil {
		db.Close()
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

	return &itemRepository{
		db: db,
	}, nil
}

func (i *itemRepository) Close() error {
    return i.db.Close()
}

// common query function
func (i *itemRepository) queryItems(ctx context.Context, query string, args ...interface{}) ([]Item, error) {
    rows, err := i.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query items: %w", err)
    }
    defer rows.Close()

    var items []Item
    for rows.Next() {
        var item Item
        if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageName); err != nil {
            return nil, fmt.Errorf("failed to scan item: %w", err)
        }
        items = append(items, item)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error during iteration: %w", err)
    }

    return items, nil
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
    if item == nil {
        return errInvalidInput
    }

    stmt, err := i.db.PrepareContext(ctx, `
        INSERT INTO items (name, category, image_name)
        VALUES (?, ?, ?)
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()

    result, err := stmt.ExecContext(ctx, item.Name, item.Category, item.ImageName)
    if err != nil {
        return fmt.Errorf("failed to insert item: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert id: %w", err)
    }
    item.ID = int(id)

    return nil
}

// List returns all items from the repository.
func (i *itemRepository) List(ctx context.Context) ([]Item, error) {
    return i.queryItems(ctx, `
        SELECT i.id, i.name, i.category, i.image_name 
        FROM items i 
        JOIN categories c ON i.category_id = c.id
    `)
}

// Get returns a specific item from the repository.
func (i *itemRepository) Get(ctx context.Context, id string) (*Item, error) {
    if id == "" {
        return nil, errInvalidInput
    }

    var item Item
    err := i.db.QueryRowContext(ctx, `
        SELECT i.id, i.name, c.name AS category, i.image_name 
        FROM items i 
        INNER JOIN categories c ON i.category_id = c.id 
        WHERE i.id = ?
    `, id).Scan(&item.ID, &item.Name, &item.Category, &item.ImageName)

    if err == sql.ErrNoRows {
        return nil, errItemNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get item: %w", err)
    }

    return &item, nil
}

// Search searches items containing the given keyword in their name.
func (i *itemRepository) Search(ctx context.Context, keyword string) ([]Item, error) {
    if keyword == "" {
        return nil, errInvalidInput
    }

    return i.queryItems(ctx, `
        SELECT i.id, i.name, c.name AS category, i.image_name 
        FROM items i 
        JOIN categories c ON i.category_id = c.id 
        WHERE i.name LIKE ?
    `, "%"+keyword+"%")
}

// returns the category ID for a given category name
func (i *itemRepository) GetCategoryID(ctx context.Context, categoryName string) (int, error) {
	var id int
	err := i.db.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = ?", categoryName).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("category not found: %s", categoryName)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get category id: %w", err)
	}
	return id, nil
}

// returns the category name for a given category id	
func (i *itemRepository) GetCategoryName(ctx context.Context, categoryID int) (string, error) {
    var name string
    err := i.db.QueryRowContext(ctx, "SELECT name FROM categories WHERE id = ?", categoryID).Scan(&name)
    if err == sql.ErrNoRows {
        return "", fmt.Errorf("category not found with id: %d", categoryID)
    }
    if err != nil {
        return "", fmt.Errorf("failed to get category name: %w", err)
    }
    return name, nil
}