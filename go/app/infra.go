package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

//custom error
var errImageNotFound = errors.New("image not found")
var errItemNotFound = errors.New("item not found") 

type Item struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
	// STEP 4-2: add a category field:
	Category  string `db:"category" json:"category"`
	// STEP 4-4: add an image_name field:
	ImageName string `db:"image_name" json:"image_name"` 
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error //insert an item
	List(ctx context.Context) ([]Item, error) //get all items
	Get(ctx context.Context, id string) (*Item, error) //get an item by id
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
	// filePath is the absolute path to the JSON file
	filePath string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() (ItemRepository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	fileName := "items.json"
	filePath := filepath.Join(cwd, fileName)

	// check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create items file: %w", err)
		}
		defer file.Close()

		// write initial data
		initialData := ItemsData{Items: []Item{}}
		if err := json.NewEncoder(file).Encode(initialData); err != nil {
			return nil, fmt.Errorf("failed to write initial data: %w", err)
		}
	}

	return &itemRepository{
		fileName: fileName,
		filePath: filePath,
	}, nil
}
//追加
// JSONファイルの内容をパースするための構造体
type ItemsData struct {
	Items []Item `json:"items"`
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	// STEP 4-2: add an implementation to store an item

	// open the JSON file
	file, err := os.OpenFile(i.filePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open items file: %w", err)
	}
	defer file.Close()

	// read the content of the JSON file
	var data ItemsData
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil && err != io.EOF {
		return err
	}
	// IDを設定
	item.ID = len(data.Items) + 1

	// 新しいアイテムを追加
	data.Items = append(data.Items, *item)

	// ファイルを再度開いて内容をリセット
	file.Seek(0, 0)
	file.Truncate(0)

	// 更新されたアイテムリストを書き込む
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // 見やすくする
	err = encoder.Encode(data)
	//err = json.NewEncoder(file).Encode(items)
	if err != nil {
		return err
	}


	return nil
}

//追加
// List returns all items from the repository.
func (i *itemRepository) List(ctx context.Context) ([]Item, error) {
	
	// open the JSON file
    file, err := os.OpenFile(i.filePath, os.O_RDONLY, 0644)
    if err != nil {
        return nil, fmt.Errorf("failed to open items file: %w", err)
    }
    defer file.Close()

	// read the content of the JSON file
	var data ItemsData
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return data.Items, nil
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image

	// ファイルを書き込みモードで開く
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// 画像データを書き込む
	_, err = file.Write(image)
	if err != nil {
		return err
	}

	return nil
}

//追加
// Get returns a specific item from the repository.
func (i *itemRepository) Get(ctx context.Context, id string) (*Item, error) {

	// open the JSON file
	file, err := os.OpenFile(i.filePath, os.O_RDONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("items file not found: %w", err)
		}
		return nil, fmt.Errorf("failed to open items file: %w", err)
	}
	defer file.Close()

	// read the content of the JSON file
	var data ItemsData
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// IDを整数に変換
	itemID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid item id: %s", id)
	}

	// 指定されたIDの商品を検索
	for _, item := range data.Items {
		if item.ID == itemID {
			return &item, nil
		}
	}

	return nil, errItemNotFound
}
