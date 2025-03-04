package app

import (
	"context"
	"errors"
	"encoding/json"
	"os"
	"io"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID   int    `db:"id" json:"-"`
	Name string `db:"name" json:"name"`
	// STEP 4-2: add a category field:
	Category string `db:"category" json:"category"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	List(ctx context.Context) ([]Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}

//JSONファイルの内容をパースするための構造体
type ItemsData struct {
	Items []Item `json:"items"`
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
    // STEP 4-1: add an implementation to store an item

	cwd, err := os.Getwd()
    if err != nil {
        return err
    }

	// 絶対パスを作成
    filePath := cwd + "/" + i.fileName

    // JSONファイルを開く (なければ作成)
    file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
    if err != nil {
        return err
    }
    defer file.Close()

	// JSONの中身を読み取る
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

// List returns all items from the repository.
func (i *itemRepository) List(ctx context.Context) ([]Item, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// 絶対パスを作成
	filePath := cwd + "/" + i.fileName

	// JSONファイルを開く
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0755)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// JSONの中身を読み取る
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