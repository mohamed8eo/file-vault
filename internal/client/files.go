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

	fmt.Printf("%-36s  %-30s  %s\n", "ID", "NAME", "SIZE")
	fmt.Println("--------------------------------------------------------------------------------------------")
	for _, f := range files {
		fmt.Printf("%-36v  %-30v  %s\n", f["id"], f["file_name"], f["created_at"])
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

	req, err := http.NewRequest("POST", baseURL+"/upload", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+LoadToken())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload file")
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("uploaded successfully — id: %v\n", result["id"])
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file not found")
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	url := result["file_url"]
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

	exec.Command(cmd, url).Start()
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
