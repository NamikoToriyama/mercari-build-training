package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir        = "images"
	ItemsJsonPath = "./../db/items.json"
)

type Response struct {
	Message string `json:"message"`
}

type Item struct {
	Name          string `json:"name"`
	Category      string `json:"category"`
	ImageFilename string `json:imageFilename`
}

type Items struct {
	Items []Item `json:"items"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func listItems() (*Items, error) {
	// jsonファイルの読み込み
	file, err := os.OpenFile(ItemsJsonPath, os.O_RDWR|os.O_CREATE, 0644) // os.openでも良い。書き込み権限があるか確認
	if err != nil {
		return nil, fmt.Errorf("Error opening file:", err)
	}
	defer file.Close()

	// jsonファイルをdecode
	var currentItems Items
	err = json.NewDecoder(file).Decode(&currentItems)
	if err != nil {
		return nil, fmt.Errorf("Error decoding JSON:", err)
	}
	return &currentItems, nil
}

func hashString(input string) string {
	// https://pkg.go.dev/crypto/sha256
	hash := sha256.New()
	hash.Write([]byte(input))
	hashed := hash.Sum(nil)
	// 文字列にする
	return hex.EncodeToString(hashed)
}

func storeImage(image *multipart.FileHeader, imageName string) error {
	// 取得した画像をファイルに保存
	file, err := image.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a new file for the uploaded image
	filePath := path.Join(ImgDir, imageName+".jpg")
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy the uploaded image data to the new file
	_, err = io.Copy(out, file)
	if err != nil {
		return err
	}

	return nil
}

// add items to json
// {"items": [{"name": "jacket", "category": "fashion"}, ...]}
// point: errorの処理をちゃんとやること
func addItemToJson(c echo.Context, item *Item) error {
	currentItems, err := listItems()
	if err != nil {
		return err
	}
	// itemを追加
	currentItems.Items = append(currentItems.Items, *item)

	// 書き込み用にファイルを開く
	file, err := os.Create(ItemsJsonPath)
	if err != nil {
		return fmt.Errorf("Error opening file for writing:", err)
	}
	defer file.Close()

	// jsonファイルに書き込み
	err = json.NewEncoder(file).Encode(currentItems)
	if err != nil {
		return fmt.Errorf("Error encoding JSON:", err)
	}
	return nil
}

func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("failed c.FormFile: %s", err.Error()))
	}

	// ファイル名をhash化
	hashedImageName := hashString(image.Filename)

	// 画像として保存
	if err := storeImage(image, hashedImageName); err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("failed storeImage: %s", err.Error()))
	}

	c.Logger().Infof("Receive item: %s category: %s", name, category)
	item := &Item{
		Name:          name,
		Category:      category,
		ImageFilename: hashedImageName,
	}

	if err := addItemToJson(c, item); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}

	message := fmt.Sprintf("item received: %s category received: %s image received: %s", name, category, hashedImageName)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	currentItems, err := listItems()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, currentItems)
}

func getItem(c echo.Context) error {
	items, err := listItems()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("failed GetItems: %s", err.Error()))
	}

	str_id := c.Param("id")
	// 文字列を整数に変換
	id, err := strconv.Atoi(str_id)
	fmt.Println("Error opening file:", id, len(items.Items))
	if err != nil || id < 0 || len(items.Items) < id {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	return c.JSON(http.StatusOK, items.Items[id-1])
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items", getItems)
	e.GET("/items/:id", getItem)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":8000"))
}
