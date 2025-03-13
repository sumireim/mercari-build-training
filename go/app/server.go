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
	"context"
	"io"
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
	// Set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Set up CORS settings
	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// Set up handlers
	itemRepo, err := NewItemRepository()
	if err != nil {
		slog.Error("failed to create item repository: ", "error", err)
		return 1
	}

	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /items/{id}", h.GetItemDetail)
	mux.HandleFunc("GET /search", h.Search)

	// Start the server
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
	Category string `form:"category"` // Category of the item
	Image    []byte `form:"image"`    // Image data in bytes
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {
	var req = &AddItemRequest{}
	
	// Check if it's multipart/form-data
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}

		req.Name = r.FormValue("name")
		req.Category = r.FormValue("category")

		// Get the image file
		file, header, err := r.FormFile("image")
		if err != nil {
			if errors.Is(err, http.ErrMissingFile) {
				return nil, errors.New("image is required")
			}
			return nil, fmt.Errorf("failed to get image file: %w", err)
		}
		defer file.Close()

		// Check file extension (optional, but good practice)
		if !strings.HasSuffix(strings.ToLower(header.Filename), ".jpg") {
			return nil, errors.New("only .jpg files are allowed")
		}

		// Read image data
		imageData, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read image data: %w", err)
		}
		if len(imageData) == 0 {
			return nil, errors.New("image data is empty")
		}

		req.Image = imageData

	} else { // If not multipart/form-data (for testing, or if you want to support other formats)
		// parse form
		err := r.ParseForm()
		if err != nil {
			return nil, fmt.Errorf("failed to parse form: %w", err)
		}
		
		// set form values
		req.Name = r.FormValue("name")
		req.Category = r.FormValue("category")

		if imagePath := r.FormValue("image"); imagePath != "" {
			// test case
			if !strings.HasSuffix(strings.ToLower(imagePath), ".jpg") {
				return nil, errors.New("only .jpg files are allowed")
			}
	
			imageData, err := os.ReadFile(imagePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read image file: %w", err)
			}
			if len(imageData) == 0 {
				return nil, errors.New("image data is empty")
			}
			req.Image = imageData
		} 
	}
	slog.Debug("parseAddItemRequest", "name", req.Name, "category", req.Category, "image_len", len(req.Image)) 
	// Validate the request (these checks should be done regardless of Content-Type)
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.Category == "" {
		return nil, errors.New("category is required")
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
	//get category id
	/*_, err = s.itemRepo.GetCategoryID(ctx, req.Category)
    if err != nil {
        slog.Error("ailed to get category id", "error", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }*/
	// set default image name
	fileName := "default.jpg" 
	if len(req.Image) > 0 {     
		fileName, err = s.storeImage(req.Image)
		if err != nil {
			slog.Error("failed to store image: ", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	item := &Item{
		Name:       req.Name,
		Category: req.Category,
		ImageName:  fileName,
	}

	err = s.itemRepo.Insert(ctx, item)
	if err != nil {
		slog.Error("failed to store item: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the list of all items including the newly added item
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

// storeImage stores an image and returns the file path and an error if any.
// This method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
// func (s *Handlers) storeImage(image []byte) (filePath string, err error) {

func (s *Handlers) storeImage(image []byte) (string, error) {
	// Calculate SHA-256 hash
	hasher := sha256.New()
	_, err := hasher.Write(image)
	if err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}
	hashSum := hex.EncodeToString(hasher.Sum(nil))
	fileName := hashSum + ".jpg"

	// Create image file path
	filePath := filepath.Join(s.imgDirPath, fileName)

	// Skip if image with same hash already exists
	_, statErr := os.Stat(filePath)
	if statErr == nil {
		return fileName, nil
	}

	// Ensure the image directory exists
	if err := os.MkdirAll(s.imgDirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	// Save image
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

// GetItemsResponse represents the response format for the list of items
type GetItemsResponse struct {
	Items []Item `json:"items"`
}

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

// request format for getting item details
type GetItemDetailRequest struct {
	ID string // path value
}

// parses and validates the request to get an item detail.
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

// GetItemDetailResponse defines the response format for item details
type GetItemDetailResponse struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
}

// GetItemDetail is a handler to return a specific item for GET /items/{id} .
func (s *Handlers) GetItemDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := parseGetItemDetailRequest(r)
	if err != nil {
		slog.Warn("failed to parse get item detail request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get item details
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

	// Convert to response format
	resp := GetItemDetailResponse{
		Name:      item.Name,
		Category:  item.Category,
		ImageName: item.ImageName,
	}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type SearchItemsRequest struct {
	Keyword string // query value
}

// response format for search items
type SearchItemsResponse struct {
    Items []GetItemDetailResponse `json:"items"`
}

// get the keyword from the request
func parseSearchItemsRequest(r *http.Request) (*SearchItemsRequest, error) {
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		return nil, errors.New("keyword is required")
	}
	
	return &SearchItemsRequest{
		Keyword: keyword,
	}, nil
}

// Search returns a list of items containing the given keyword
func (s *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	req, err := parseSearchItemsRequest(r)
	if err != nil {
		slog.Warn("failed to parse search items request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// search items containing the given keyword
	items, err := s.itemRepo.Search(ctx, req.Keyword)
	if err != nil {
		slog.Error("failed to search items: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// convert items to response format
	var respItems []GetItemDetailResponse
	for _, item := range items {
		respItems = append(respItems, GetItemDetailResponse{
			Name:      item.Name,
			Category:  item.Category,
			ImageName: item.ImageName,
		})
	}
	
	// return the list of items containing the given keyword
	resp := SearchItemsResponse{
        Items: respItems,
    }
	
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
} 

// getCategoryID gets the category id for a given category name
func getCategoryID(ctx context.Context, itemRepo ItemRepository, categoryName string) (int, error) {
	categoryID, err := itemRepo.GetCategoryID(ctx, categoryName)
	if err != nil {
		return 0, fmt.Errorf("failed to get category id: %w", err)
	}
	return categoryID, nil
}

