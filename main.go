package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

type requestParams map[string]string

type ErrorResponse struct {
	Message     string `json:"message"`
	Description string `json:"description"`
	Error       string `json:"error"`
}

type UploadLinkResponse struct {
	Href      string `json:"href"`
	Method    string `json:"method"`
	Templated bool   `json:"templated"`
}

type Item struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type EmbeddedStruct struct {
	ItemList []Item `json:"items"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Total    int    `json:"total"`
}

type ResourcesResponse struct {
	Path     string         `json:"path"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Embedded EmbeddedStruct `json:"_embedded"`
}

type DiskClient struct {
	URL        string
	Token      string
	ClientHTTP *http.Client
}

func NewDiskClient() DiskClient {
	d := DiskClient{}

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("error loading .env file")
	}

	d.URL = os.Getenv("YANDEX_DISK_URL")
	d.Token = os.Getenv("YANDEX_OAUTH_TOKEN")
	d.ClientHTTP = &http.Client{}

	return d
}

func concatParams(params requestParams) string {
	if len(params) > 0 {
		p := "?"
		for key, val := range params {
			p += key + "=" + val + "&"
		}
		// return without tailing '&'
		return p[:len(p)-1]
	}

	return ""
}
func (d DiskClient) makeRequest(method, endpoint string, params requestParams, bodyReq io.Reader) ([]byte, error) {
	p := concatParams(params)

	reqURL := d.URL + endpoint + p
	req, err := http.NewRequest(method, reqURL, bodyReq)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "OAuth "+d.Token)

	resp, err := d.ClientHTTP.Do(req)
	if err != nil {
		return nil, err
	}

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		errResp := ErrorResponse{}
		err = json.Unmarshal(bodyResp, &errResp)
		if err != nil {
			return nil, fmt.Errorf("occure error during unmarshal response error from disk API\n%v", err)
		}

		return nil, fmt.Errorf("occure error during request to disk API\nrequest url%v\n%v ", reqURL, errResp)
	}

	return bodyResp, nil
}

func (d DiskClient) getUploadURL(pathOnDisk string) (string, error) {
	params := requestParams{"path": pathOnDisk}

	bodyResp, err := d.makeRequest("GET", "/resources/upload", params, nil)
	if err != nil {
		return "", err
	}

	link := UploadLinkResponse{}
	err = json.Unmarshal(bodyResp, &link)
	if err != nil {
		return "", err
	}

	if link.Href == "" {
		return "", fmt.Errorf("url for upload file is empty")
	}

	return link.Href, nil
}

func (d DiskClient) uploadFile(uploadURL, filePath string) error {
	args := []string{
		"-X",
		"PUT",
		uploadURL,
		"--upload-file",
		filePath,
	}

	cmd := exec.Command("curl", args...)
	stdout, err := cmd.Output()

	if err != nil {
		return err
	}

	if len(stdout) > 0 {
		return nil
	}

	return fmt.Errorf("stdout doesn't emplty - %v", string(stdout))
}

func main() {

	d := NewDiskClient()

	uploadURL, err := d.getUploadURL("/test_folder/heh")
	if err != nil {
		log.Fatal(err)
	}

	err = d.uploadFile(uploadURL, "heh")
	if err != nil {
		log.Fatal(err)
	}

	// reqUrl := diskUrl + "/resources?path=/"

	// req, err := http.NewRequest("GET", reqUrl, nil)
	// if err != nil {
	// 	fmt.Print(err)
	// }
	// req.Header.Set("Authorization", "OAuth "+token)

	// resp, err := client.Do(req)
	// if err != nil {
	// 	fmt.Print(err)
	// }

	// if resp.StatusCode > 299 || resp.StatusCode < 200 {
	// 	fmt.Printf("bad status code - %v", resp.StatusCode)
	// }
	// fmt.Print(resp.Status)
	// defer resp.Body.Close()
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Print(err)
	// }

	// var fold ResourcesResponse

	// if err = json.Unmarshal(body, &fold); err != nil {
	// 	fmt.Print(err)
	// }

	// fmt.Print(fold)
	// files, err := os.ReadDir(directory)
	// if err != nil {
	// 	fmt.Print(err)
	// }

	// for _, file := range files {
	// 	fmt.Println(file.Name())
	// }
}
