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

var loadingChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func loadingSpinner(done chan bool) {
	i := 0
	for {
		select {
		case <-done:
			fmt.Print("\r")
			return
		default:
			fmt.Printf("\r%s Loading...", loadingChars[i%len(loadingChars)])
			time.Sleep(50 * time.Millisecond)
			i++
		}
	}
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %c", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func ListFiles(limit, page, offset int) error {
	if LoadToken() == "" {
		return fmt.Errorf("not logged in, run: file-vault auth login")
	}

	if page > 0 && offset == 0 {
		offset = (page - 1) * limit
	}

	done := make(chan bool, 1)
	go loadingSpinner(done)

	url := fmt.Sprintf("%s/files?limit=%d&offset=%d", baseURL, limit, offset)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		done <- true
		return err
	}
	req.Header.Set("Authorization", "Bearer "+LoadToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		done <- true
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := refreshAccessToken(); err != nil {
			done <- true
			return err
		}
		req.Header.Set("Authorization", "Bearer "+LoadToken())
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			done <- true
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		done <- true
		return fmt.Errorf("failed to list files")
	}

	var files []map[string]any
	json.NewDecoder(resp.Body).Decode(&files)

	done <- true
	time.Sleep(100 * time.Millisecond)

	if len(files) == 0 {
		fmt.Println("no files found")
		return nil
	}

	idLen := 36
	nameLen := 25
	sizeLen := 20
	dateLen := 22

	header := fmt.Sprintf("%-*s | %-*s | %-*s | %-*s", idLen, "id", nameLen, "name", sizeLen, "Size", dateLen, "created_at")
	separator := strings.Repeat("-", len(header))
	fmt.Println(header)
	fmt.Println(separator)

	for _, f := range files {
		id, _ := f["id"].(string)
		name, _ := f["file_name"].(string)
		var sizeStr string
		if size, ok := f["file_size"].(float64); ok && size > 0 {
			sizeStr = formatFileSize(int64(size))
		} else {
			sizeStr = "-"
		}
		dateStr, _ := f["created_at"].(string)

		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			t = time.Now()
		}
		date := t.Format("Jan 2, 2006 at 3:04 PM")

		if len(name) > nameLen {
			name = name[:nameLen-3] + "..."
		}
		fmt.Printf("%-*s | %-*s | %-*s | %-*s\n", idLen, id, nameLen, name, sizeLen, sizeStr, dateLen, date)
	}

	fmt.Println()
	if len(files) == limit {
		fmt.Printf("Showing page %d (offset: %d)\n", page, offset)
		fmt.Println("Use --page or --offset to see more")
	}

	return nil
}

func showUploadProgress(stop chan bool) {
	frames := []string{"▁▂▃▄▅▆▇█", "█▇▆▅▄▃▂▁", "▓▒░", "░▒▓"}
	i := 0
	for {
		select {
		case <-stop:
			return
		default:
			fmt.Printf("\r  ↔ %s", frames[i%len(frames)])
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
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

	stop := make(chan bool, 1)
	go showUploadProgress(stop)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		stop <- true
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
		stop <- true
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := refreshAccessToken(); err != nil {
			stop <- true
			return err
		}
		resp, err = makeReq()
		if err != nil {
			stop <- true
			return err
		}
		defer resp.Body.Close()
	}

	stop <- true
	fmt.Print("\r")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload file")
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	id, ok := result["id"].(string)
	if !ok {
		fmt.Println("✓ Uploaded successfully")
		return nil
	}
	fmt.Printf("✓ Uploaded — id: %s\n", id)
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
		return fmt.Errorf("file URL is missing")
	}

	fmt.Println("url:", url)

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

	err = exec.Command(cmd, url).Start()
	if err != nil {
		fmt.Printf("Failed to open browser. Please open manually: %s\n", url)
		return err
	}
	return nil
}

func SearchFiles(query string, limit int) error {
	if LoadToken() == "" {
		return fmt.Errorf("not logged in, run: file-vault auth login")
	}

	url := fmt.Sprintf("%s/files/search?q=%s&limit=%d", baseURL, query, limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+LoadToken())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := refreshAccessToken(); err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+LoadToken())
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to search files")
	}

	var files []map[string]any
	json.NewDecoder(resp.Body).Decode(&files)

	if len(files) == 0 {
		fmt.Println("no files found")
		return nil
	}

	idLen := 36
	nameLen := 25
	sizeLen := 20
	dateLen := 22

	fmt.Printf("Searching for: \"%s\"\n\n", query)

	header := fmt.Sprintf("%-*s | %-*s | %-*s | %-*s", idLen, "id", nameLen, "name", sizeLen, "Size", dateLen, "created_at")
	separator := strings.Repeat("-", len(header))
	fmt.Println(header)
	fmt.Println(separator)

	for _, f := range files {
		id, _ := f["id"].(string)
		name, _ := f["file_name"].(string)
		var sizeStr string
		if size, ok := f["file_size"].(float64); ok && size > 0 {
			sizeStr = formatFileSize(int64(size))
		} else {
			sizeStr = "-"
		}
		dateStr, _ := f["created_at"].(string)

		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			t = time.Now()
		}
		date := t.Format("Jan 2, 2006 at 3:04 PM")

		if len(name) > nameLen {
			name = name[:nameLen-3] + "..."
		}
		fmt.Printf("%-*s | %-*s | %-*s | %-*s\n", idLen, id, nameLen, name, sizeLen, sizeStr, dateLen, date)
	}

	fmt.Println()
	fmt.Printf("Found %d file(s)\n", len(files))
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