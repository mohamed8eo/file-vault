package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

const baseURL = "http://localhost:3000"

type config struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func saveTokens(accessToken, refreshToken string) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".file-vault")
	os.MkdirAll(dir, 0o700)
	data, _ := json.Marshal(config{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0o600)
}

func loadConfig() config {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".file-vault", "config.json"))
	if err != nil {
		return config{}
	}
	var cfg config
	json.Unmarshal(data, &cfg)
	return cfg
}

func LoadToken() string {
	return loadConfig().AccessToken
}

func refreshAccessToken() error {
	cfg := loadConfig()
	if cfg.RefreshToken == "" {
		return fmt.Errorf("session expired, please login again")
	}

	req, err := http.NewRequest("POST", baseURL+"/auth/refresh", nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: cfg.RefreshToken,
	})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session expired, please login again")
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	newAccess := result["access_token"]
	if newAccess == "" {
		return fmt.Errorf("session expired, please login again")
	}

	// extract new refresh token from cookie if present
	newRefresh := cfg.RefreshToken
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "refresh_token" {
			newRefresh = cookie.Value
			break
		}
	}

	return saveTokens(newAccess, newRefresh)
}

func AuthRequest(method, path string, body []byte) (*http.Response, error) {
	makeReq := func() (*http.Response, error) {
		var req *http.Request
		var err error
		if body != nil {
			req, err = http.NewRequest(method, baseURL+path, bytes.NewBuffer(body))
		} else {
			req, err = http.NewRequest(method, baseURL+path, nil)
		}
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+LoadToken())
		req.Header.Set("Content-Type", "application/json")
		return http.DefaultClient.Do(req)
	}

	resp, err := makeReq()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := refreshAccessToken(); err != nil {
			return nil, err
		}
		return makeReq()
	}

	return resp, nil
}

func Login(email, password string) error {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid email or password")
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	accessToken := result["access_token"]
	if accessToken == "" {
		return fmt.Errorf("login failed: no token returned")
	}

	// extract refresh token from cookie
	var refreshToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "refresh_token" {
			refreshToken = cookie.Value
			break
		}
	}

	if err := saveTokens(accessToken, refreshToken); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("logged in successfully")
	return nil
}

func Register(name, email, password string) error {
	body, _ := json.Marshal(map[string]string{
		"name":     name,
		"email":    email,
		"password": password,
	})
	resp, err := http.Post(baseURL+"/auth/sign-up", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed")
	}

	fmt.Println("registered successfully, now run: file-vault auth login")
	return nil
}

func Logout() error {
	cfg := loadConfig()

	req, _ := http.NewRequest("POST", baseURL+"/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "refresh_token",
		Value: cfg.RefreshToken,
	})
	http.DefaultClient.Do(req)

	home, _ := os.UserHomeDir()
	os.Remove(filepath.Join(home, ".file-vault", "config.json"))
	fmt.Println("logged out")
	return nil
}

func Status() {
	cfg := loadConfig()
	if cfg.AccessToken == "" {
		fmt.Println("not logged in")
		return
	}
	fmt.Println("logged in")
}
