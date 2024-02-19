package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "images"
)

type Response struct {
	Message string `json:"message"`
}

type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Items struct {
	Items []Item `json:"items"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

// add items to json
// {"items": [{"name": "jacket", "category": "fashion"}, ...]}
// point: errorの処理をちゃんとやること
func addItemToJson(c echo.Context, item *Item) error {
	// jsonファイルの読み込み
	file, err := os.OpenFile("./items.json", os.O_RDWR|os.O_CREATE, 0644) // os.openでも良い。書き込み権限があるか確認
	if err != nil {
		return fmt.Errorf("Error opening file:", err)
	}
	defer file.Close()

	// jsonファイルをdecode
	var currentItems Items
	err = json.NewDecoder(file).Decode(&currentItems)
	if err != nil {
		return fmt.Errorf("Error decoding JSON:", err)
	}

	// itemを追加
	currentItems.Items = append(currentItems.Items, *item)

	// 書き込み用にファイルを開く
	file, err = os.Create("./items.json")
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
	c.Logger().Infof("Receive item: %s category: %s", name, category)
	item := &Item{
		Name:     name,
		Category: category,
	}

	if err := addItemToJson(c, item); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}

	message := fmt.Sprintf("item received: %s \n category received: %s", name, category)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
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
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":8000"))
}
