package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	// Port is the port number to listen on.
	Port string
	// ImageDirPath is the path to the directory storing images.
	ImageDirPath string
}

// Run is a method to start the server.
// This method returns 0 if the server started successfully, and 1 otherwise.
func (s Server) Run() int {
	// set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	// STEP 4-6: set the log level to DEBUG
	slog.SetLogLoggerLevel(slog.LevelDebug)
	
	// set up CORS settings
	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// STEP 5-1: set up the database connection

	// set up handlers
	itemRepo, err := NewItemRepository()
	if err != nil {
		slog.Error("failed to create item repository: ", "error", err)
		return 1
	}
	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /items", h.GetItems) //GET /items エンドポイントのハンドラを追加
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /items/{id}", h.GetItemDetail) //GET /items/<item_id>エンドポイントを作成

	// start the server
	slog.Info("http server started on", "port", s.Port)
	err = http.ListenAndServe(":"+s.Port, simpleCORSMiddleware(simpleLoggerMiddleware(mux), frontURL, []string{"GET", "HEAD", "POST", "OPTIONS"}))
	if err != nil {
		slog.Error("failed to start server: ", "error", err)
		return 1
	}

	return 0
}

type Handlers struct {
	// imgDirPath is the path to the directory storing images.
	imgDirPath string
	itemRepo   ItemRepository
}

type HelloResponse struct {
	Message string `json:"message"`
}

// Hello is a handler to return a Hello, world! message for GET / .
func (s *Handlers) Hello(w http.ResponseWriter, r *http.Request) {
	resp := HelloResponse{Message: "Hello, world!"}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type AddItemRequest struct {
	Name     string `form:"name"`
	Category string `form:"category"` // STEP 4-2: add a category field:
	Image    []byte `form:"image"`    // STEP 4-4: add an image field
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {

	req := &AddItemRequest{
		Name:     r.FormValue("name"),
		Category: r.FormValue("category"),
		// STEP 4-2: add a category field:
	}

	// make sure to parse the multipart form first
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}
	// validate the request
	// 画像ファイルを取得
	file, header, err := r.FormFile("image")
	if err != nil {
		return nil, fmt.Errorf("failed to get image file: %w", err)
	}
	defer file.Close()

	// 画像がjpgか確認
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".jpg") {
		return nil, errors.New("only .jpg files are allowed")
	}

	// 画像データを読み込む
	imageData := make([]byte, header.Size)
	_, err = file.Read(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}
	req.Image = imageData

	// validate the request
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	// STEP 4-2: validate the category field:
	if req.Category == "" {
		return nil, errors.New("category is required")
	}

	// STEP 4-4: validate the image field
	if len(req.Image) == 0 {
		return nil, errors.New("image is required")
	}

	return req, nil
}

// AddItem is a handler to add a new item for POST /items .
func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := parseAddItemRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// STEP 4-4: uncomment on adding an implementation to store an image
	fileName, err := s.storeImage(req.Image)
	if err != nil {
		slog.Error("failed to store image: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	item := &Item{
		Name:     req.Name,
		Category: req.Category,
		// STEP 4-2: add a category field:
		// STEP 4-4: add an image field
		ImageName: fileName,
	}
	//step4-4によりコメントアウト
	//message := fmt.Sprintf("item received: %s", item.Name)
	//slog.Info(message)

	// STEP 4-2: add an implementation to store an item
	err = s.itemRepo.Insert(ctx, item)
	if err != nil {
		slog.Error("failed to store item: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//resp := AddItemResponse{Message: message}//下記に変更
	// 登録した商品を含む全商品リストを返す
	items, err := s.itemRepo.List(ctx)
	if err != nil {
		slog.Error("failed to get items: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//追加ここまで

	resp := GetItemsResponse{Items: items}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// storeImage stores an image and returns the file path and an error if any.
// this method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
// func (s *Handlers) storeImage(image []byte) (filePath string, err error) {
// STEP 4-4: add an implementation to store an image
// TODO:
// - calc hash sum
// - build image file path
// - check if the image already exists
// - store image
// - return the image file path
func (s *Handlers) storeImage(image []byte) (string, error) {
	// SHA-256ハッシュを計算
	hasher := sha256.New()
	_, err := hasher.Write(image)
	if err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}
	hashSum := hex.EncodeToString(hasher.Sum(nil))
	fileName := hashSum + ".jpg"

	// 画像ファイルのパスを作成
	filePath := filepath.Join(s.imgDirPath, fileName)

	// 既に同じハッシュの画像が存在する場合は保存をスキップ
	_, statErr := os.Stat(filePath)
	if statErr == nil {
		return fileName, nil
	}

	// 画像を保存
	err = os.WriteFile(filePath, image, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return fileName, nil
}

type GetImageRequest struct {
	FileName string // path value
}

// parseGetImageRequest parses and validates the request to get an image.
func parseGetImageRequest(r *http.Request) (*GetImageRequest, error) {
	req := &GetImageRequest{
		FileName: r.PathValue("filename"), // from path parameter
	}

	// validate the request
	if req.FileName == "" {
		return nil, errors.New("filename is required")
	}

	return req, nil
}

// GetImage is a handler to return an image for GET /images/{filename} .
// If the specified image is not found, it returns the default image.
func (s *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetImageRequest(r)
	if err != nil {
		slog.Warn("failed to parse get image request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imgPath, err := s.buildImagePath(req.FileName)
	if err != nil {
		if !errors.Is(err, errImageNotFound) {
			slog.Warn("failed to build image path: ", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// when the image is not found, it returns the default image without an error.
		slog.Debug("image not found", "filename", imgPath)
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}

	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}

// buildImagePath builds the image path and validates it.
func (s *Handlers) buildImagePath(imageFileName string) (string, error) {
	imgPath := filepath.Join(s.imgDirPath, filepath.Clean(imageFileName))

	// to prevent directory traversal attacks
	rel, err := filepath.Rel(s.imgDirPath, imgPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid image path: %s", imgPath)
	}

	// validate the image suffix
	if !strings.HasSuffix(imgPath, ".jpg") {
		return "", fmt.Errorf("image path does not end with .jpg: %s", imgPath)
	}

	// check if the image exists
	_, err = os.Stat(imgPath)
	if err != nil {
		return imgPath, errImageNotFound
	}

	return imgPath, nil
}

//追加
// ist メソッドの実装を追加
type GetItemsResponse struct {
	Items []Item `json:"items"`
}

//追加
// GetItems is a handler to return a list of items for GET /items .
func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, err := s.itemRepo.List(ctx)
	if err != nil {
		slog.Error("failed to get items: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := GetItemsResponse{Items: items}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//追加
type GetItemDetailRequest struct {
	ID string // path value
}

//追加
// parseGetItemDetailRequest parses and validates the request to get an item detail.
func parseGetItemDetailRequest(r *http.Request) (*GetItemDetailRequest, error) {
	req := &GetItemDetailRequest{
		ID: r.PathValue("id"), // from path parameter
	}

	// validate the request
	if req.ID == "" {
		return nil, errors.New("item id is required")
	}

	return req, nil
}

//追加
// GetItemDetailResponse は商品詳細のレスポンス形式を定義
type GetItemDetailResponse struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
}

//追加
// GetItemDetail is a handler to return a specific item for GET /items/{id} .
func (s *Handlers) GetItemDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := parseGetItemDetailRequest(r)
	if err != nil {
		slog.Warn("failed to parse get item detail request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 商品の詳細情報を取得
	item, err := s.itemRepo.Get(ctx, req.ID)
	if err != nil {
		if errors.Is(err, errItemNotFound) {
			http.Error(w, "item not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to get item: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//err = json.NewEncoder(w).Encode(item)
	// レスポンス用の構造体に変換
	resp := GetItemDetailResponse{
		Name:      item.Name,
		Category:  item.Category,
		ImageName: item.ImageName,
	}
	//ここまで追加

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//ここまで
