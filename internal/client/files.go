package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func ListFiles() error {
	if LoadToken() == "" {
		return fmt.Errorf("not logged in, run: file-vault auth login")
	}

	resp, err := AuthRequest("GET", "/files", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to list files")
	}

	var files []map[string]any
	json.NewDecoder(resp.Body).Decode(&files)

	if len(files) == 0 {
		fmt.Println("no files found")
		return nil
	}

	idLen := 36
	nameLen := 25
	dateLen := 22

	header := fmt.Sprintf("%-*s | %-*s | %-*s", idLen, "id", nameLen, "name", dateLen, "created_at")
	separator := strings.Repeat("-", len(header))
	fmt.Println(header)
	fmt.Println(separator)

	for _, f := range files {
		id := f["id"].(string)
		name := f["file_name"].(string)
		dateStr := f["created_at"].(string)

		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			t = time.Now()
		}
		date := t.Format("Jan 2, 2006 at 3:04 PM")

		if len(name) > nameLen {
			name = name[:nameLen-3] + "..."
		}
		fmt.Printf("%-*s | %-*s | %-*s\n", idLen, id, nameLen, name, dateLen, date)
	}
	return nil
}

func UploadFile(path string) error {
	if LoadToken() == "" {
		return fmt.Errorf("not logged in, run: file-vault auth login")
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return err
	}
	io.Copy(part, file)
	writer.Close()

	makeReq := func() (*http.Response, error) {
		req, err := http.NewRequest("POST", baseURL+"/upload", &buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+LoadToken())
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return http.DefaultClient.Do(req)
	}

	resp, err := makeReq()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := refreshAccessToken(); err != nil {
			return err
		}
		resp, err = makeReq()
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload file")
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	id, ok := result["id"].(string)
	if !ok {
		fmt.Println("uploaded successfully")
		return nil
	}
	fmt.Printf("uploaded successfully — id: %s\n", id)
	return nil
}

func GetFile(id string) error {
	if LoadToken() == "" {
		return fmt.Errorf("not logged in, run: file-vault auth login")
	}

	resp, err := AuthRequest("GET", "/files/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("file not found or access denied")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file not found")
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	url, ok := result["file_url"]
	if !ok || url == "" {
		return fmt.Errorf("file URL is missing or empty in the server response")
	}

	fmt.Println("download url:", url)

	fmt.Println("\nopening in browser...")
	var cmd string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "start"
	default:
		cmd = "xdg-open"
	}

	// Attempt to open the URL
	err = exec.Command(cmd, url).Start()
	if err != nil {
		fmt.Printf("Failed to open browser. Please open the file manually: %s\n", url)
		return err
	}
	return nil
}

func DeleteFile(id string) error {
	if LoadToken() == "" {
		return fmt.Errorf("not logged in, run: file-vault auth login")
	}

	resp, err := AuthRequest("DELETE", "/files/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete file")
	}

	fmt.Println("file deleted successfully")
	return nil
}
