package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Folder struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func main() {
	client := &http.Client{}

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("error loading .env file")
	}

	// directory := os.Getenv("DIR_PATH")
	diskUrl := os.Getenv("YANDEX_DISK_URL")
	token := os.Getenv("YANDEX_OAUTH_TOKEN")

	reqUrl := diskUrl + "/resources?path=/"

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		fmt.Print(err)
	}
	req.Header.Set("Authorization", "OAuth "+token)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Print(err)
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		fmt.Printf("bad status code - %v", resp.StatusCode)
	}
	fmt.Print(resp.Status)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err)
	}

	var fold Folder

	if err = json.Unmarshal(body, &fold); err != nil {
		fmt.Print(err)
	}

	fmt.Print(fold)
	// files, err := os.ReadDir(directory)
	// if err != nil {
	// 	fmt.Print(err)
	// }

	// for _, file := range files {
	// 	fmt.Println(file.Name())
	// }
}
