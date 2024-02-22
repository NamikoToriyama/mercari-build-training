package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	_ "github.com/mattn/go-sqlite3"
)

const (
	DB_PATH = "../db/mercari.sqlite3"

	ImgDir        = "images"
	ItemsJsonPath = "./../db/items.json"
)

func HashString(input string) string {
	// https://pkg.go.dev/crypto/sha256
	hash := sha256.New()
	hash.Write([]byte(input))
	hashed := hash.Sum(nil)
	// 文字列にする
	return hex.EncodeToString(hashed)
}

func StoreImage(image *multipart.FileHeader, imageName string) error {
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

type Response struct {
	Message string `json:"message"`
}

type Item struct {
	ID            int32
	Name          string `json:"name"`
	Category      string `json:"category"`
	ImageFileName string `json:"imageFileName"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func writeItemToDB(item *Item) error {
	// sqlite3 DB open
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return err
	}
	defer db.Close()

	// itemを追加
	stmt, err := db.Prepare("insert into items(name, category, image_name) values(?, ?, ?);")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(item.Name, item.Category, item.ImageFileName)
	if err != nil {
		log.Fatal(err)
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
	hashedImageName := HashString(image.Filename)

	// 画像として保存
	if err := StoreImage(image, hashedImageName); err != nil {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("failed storeImage: %s", err.Error()))
	}

	c.Logger().Infof("Receive item: %s category: %s", name, category)
	item := &Item{
		Name:          name,
		Category:      category,
		ImageFileName: hashedImageName,
	}

	if err := writeItemToDB(item); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: "writeItemToDB" + err.Error()})
	}

	message := fmt.Sprintf("item received: %s category received: %s image received: %s", name, category, hashedImageName)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM items")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFileName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, items)
}

func getItem(c echo.Context) error {
	str_id := c.Param("id")
	// 文字列を整数に変換
	id, err := strconv.Atoi(str_id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}
	defer db.Close()

	var item Item
	err = db.QueryRow("SELECT * FROM items where id = ?", id).Scan(&item.ID, &item.Name, &item.Category, &item.ImageFileName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, item)
}

func searchItem(c echo.Context) error {
	keyword := c.QueryParam("keyword")

	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM items WHERE name LIKE CONCAT('%', ?, '%')", keyword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.ImageFileName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, Response{Message: err.Error()})
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, items)
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
	e.GET("/search", searchItem)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":8000"))
}
