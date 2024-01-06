package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const memoryPath = "/api/memory"

//go:embed static/*
var staticContent embed.FS

type ImageDetails struct {
	FullPath string
	UUID     string
}

type ImageDb struct {
	ImagePathsById map[string]ImageDetails
	ImagePaths     []ImageDetails
}

type CurrentImagePayload struct {
	URL string `json:"url"`
}

func main() {

	indexHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		indexHtml, err := staticContent.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHtml)
	}

	const defaultImagePath = "/mnt/sean-documents/art concept ai/memories_of_mount_pleasant"
	imagePath := os.Getenv("IMAGE_PATH")
	if imagePath == "" {
		imagePath = defaultImagePath
	}
	db, err := buildImageList(imagePath)
	if err != nil {
		log.Fatalf("Unable to build image list: %s", err)
	}
	numImages := len(db.ImagePaths)
	if numImages == 0 {
		log.Fatalf("Did not find any images in location: %s", imagePath)
	}
	currentMemoryHandler := func(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
		sinceEpoch := time.Now().Unix()
		const secondsPerImage = 5
		period := sinceEpoch / secondsPerImage
		imageIndex := int(period) % numImages
		currImage := db.ImagePaths[imageIndex]
		payload := CurrentImagePayload{
			URL: getMemoryUrl(currImage),
		}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling json payload: %s", err)
			http.Error(writer, "error getting image", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		setNoCacheHeaders(writer)
		writer.Write(jsonData)
	}

	memoryHandler := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("uuid")
		if len(id) != 36 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		details, exists := db.ImagePathsById[id]
		if !exists {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		file, err := os.OpenFile(details.FullPath, os.O_RDONLY, 0644)
		if err != nil {
			log.Printf("Error opening file %s: %s", details.FullPath, err)
			http.Error(w, "Error remembering memory", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=31536000") // max-age is set to one year
		w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).UTC().Format(http.TimeFormat))
		_, err = io.Copy(w, file)
		if err != nil {
			log.Println("Error sending file to client: %s", err)
			http.Error(w, "Error recollecting memory", http.StatusInternalServerError)
			return
		}
	}

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/api/current_memory", currentMemoryHandler)
	router.GET(memoryPath+"/:uuid", memoryHandler)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func setNoCacheHeaders(writer http.ResponseWriter) {
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Pragma", "no-cache")
	expirationTime := time.Now().Add(-time.Minute)
	writer.Header().Set("Expires", expirationTime.UTC().Format(http.TimeFormat))
}

func getMemoryUrl(details ImageDetails) string {
	return memoryPath + "/" + details.UUID
}

func buildImageList(dirPath string) (ImageDb, error) {
	log.Println("Starting image list")
	db := ImageDb{
		ImagePaths:     make([]ImageDetails, 0),
		ImagePathsById: make(map[string]ImageDetails),
	}
	err := filepath.Walk(dirPath, func(fPath string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Println("Error walking file:", err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileName := filepath.Base(fPath)
		if !isPNGFile(fileName) {
			log.Printf("file is not an png: %s", fileName)
			return nil
		}
		id := (uuid.New()).String()
		details := ImageDetails{
			FullPath: fPath,
			UUID:     id,
		}
		db.ImagePathsById[id] = details
		db.ImagePaths = append(db.ImagePaths, details)
		return nil
	})
	if err != nil {
		return ImageDb{}, fmt.Errorf("Error listing images: %s", err)
	}
	numImages := len(db.ImagePaths)
	log.Printf("Built collection of %d images", numImages)
	return db, nil

}

func isPNGFile(filename string) bool {
	// Convert the extension to lowercase for case-insensitive comparison
	ext := strings.ToLower(filepath.Ext(filename))

	// Check if the file has a .png extension
	return ext == ".png"
}
