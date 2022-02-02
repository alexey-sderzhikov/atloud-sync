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
	"go.uber.org/zap"
)

type requestParams map[string]interface{}

type ErrorResponse struct {
	Message     string `json:"message"`
	Description string `json:"description"`
	Error       string `json:"error"`
}

type LinkResponse struct {
	Href      string `json:"href"`
	Method    string `json:"method"`
	Templated bool   `json:"templated"`
}

type Item struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type EmbeddedResponse struct {
	ItemList []Item `json:"items"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Total    int    `json:"total"`
}

type ResourcesResponse struct {
	Path     string           `json:"path"`
	Name     string           `json:"name"`
	Type     string           `json:"type"`
	Embedded EmbeddedResponse `json:"_embedded"`
}

type DiskClient struct {
	URL        string
	Token      string
	ClientHTTP *http.Client
	Logger     *zap.SugaredLogger
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

	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	d.Logger = logger.Sugar()

	return d
}

func (p requestParams) toString() string {
	if len(p) > 0 {
		pString := "?"
		for key, val := range p {
			pString += key + "=" + fmt.Sprintf("%v", val) + "&"
		}
		// return without tailing '&'
		return pString[:len(pString)-1]
	}

	return ""
}

func (d DiskClient) makeRequest(method, reqURL string, params requestParams, bodyReq io.Reader) ([]byte, error) {
	p := params.toString()

	reqURL = reqURL + p
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

	bodyResp, err := d.makeRequest("GET", d.URL+"/resources/upload", params, nil)
	if err != nil {
		return "", err
	}

	link := LinkResponse{}
	err = json.Unmarshal(bodyResp, &link)
	if err != nil {
		return "", err
	}

	if link.Href == "" {
		return "", fmt.Errorf("url for upload file is empty")
	}

	return link.Href, nil
}

func (d DiskClient) uploadToDisk(uploadURL, path string) error {
	args := []string{
		"-X",
		"PUT",
		uploadURL,
		"--upload-file",
		path,
	}

	cmd := exec.Command("curl", args...)
	stdout, err := cmd.Output()

	if err != nil {
		return err
	}

	d.Logger.Infow("curl cmd is complete",
		"stdout", string(stdout),
	)
	return nil
}

func (d DiskClient) getDownloadURL(pathOnDisk string) (string, error) {
	params := requestParams{"path": pathOnDisk}

	bodyResp, err := d.makeRequest("GET", d.URL+"/resources/download", params, nil)
	if err != nil {
		return "", err
	}

	link := LinkResponse{}
	err = json.Unmarshal(bodyResp, &link)
	if err != nil {
		return "", nil
	}

	if link.Href == "" {
		return "", fmt.Errorf("url for download file is empty")
	}

	return link.Href, nil
}

func (d DiskClient) downloadFromDisk(downloadURL, localPath string) error {
	bodyReq, err := d.makeRequest("GET", downloadURL, nil, nil)
	if err != nil {
		return err
	}

	return os.WriteFile(localPath, bodyReq, 0777)
}

func (d DiskClient) UploadAllFilesInDir() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	d.Logger.Infow("find files in current dir",
		"dir", dir,
		"files count", len(files),
	)

	for _, file := range files {
		url, err := d.getUploadURL(file.Name())
		d.Logger.Infof("url for upload file%v", url)
		if err != nil {
			return err
		}

		err = d.uploadToDisk(url, file.Name())
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	d := NewDiskClient()

	d.Logger.Fatal(d.UploadAllFilesInDir())
}
