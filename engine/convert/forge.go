package convert

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultJobTimeout = 15 * time.Minute

const (
	forgeAuthURL     = "https://developer.api.autodesk.com/authentication/v2/token"
	forgeOSSBase     = "https://developer.api.autodesk.com/oss/v2"
	forgeMDBase      = "https://developer.api.autodesk.com/modelderivative/v2"
	forgeBucketFmt   = "kenga-%x"
)

// ForgeClient — клиент Autodesk Forge API для конвертации IPT/IAM в glTF.
type ForgeClient struct {
	ClientID     string
	ClientSecret string
	AccessToken  string
	ExpiresAt    time.Time
	HTTPClient   *http.Client
}

// NewForgeClient создаёт клиент из переменных окружения FORGE_CLIENT_ID, FORGE_CLIENT_SECRET.
func NewForgeClient() *ForgeClient {
	return &ForgeClient{
		ClientID:     os.Getenv("FORGE_CLIENT_ID"),
		ClientSecret: os.Getenv("FORGE_CLIENT_SECRET"),
		HTTPClient:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Configured возвращает true, если учётные данные заданы.
func (c *ForgeClient) Configured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// getToken получает OAuth2 token (client_credentials).
func (c *ForgeClient) getToken() (string, error) {
	if c.AccessToken != "" && time.Until(c.ExpiresAt) > 5*time.Minute {
		return c.AccessToken, nil
	}
	body := "grant_type=client_credentials&client_id=" + c.ClientID + "&client_secret=" + c.ClientSecret
	req, err := http.NewRequest("POST", forgeAuthURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Forge auth failed %d: %s", resp.StatusCode, string(b))
	}
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	c.AccessToken = data.AccessToken
	c.ExpiresAt = time.Now().Add(time.Duration(data.ExpiresIn) * time.Second)
	return c.AccessToken, nil
}

// UploadFile загружает файл в OSS и возвращает URN (base64).
func (c *ForgeClient) UploadFile(filePath string) (string, error) {
	token, err := c.getToken()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	objectName := filepath.Base(filePath)
	bucketKey := fmt.Sprintf(forgeBucketFmt, time.Now().UnixNano()%0xFFFFFF)

	// Создаём bucket
	createReq, _ := http.NewRequest("POST", forgeOSSBase+"/buckets", bytes.NewBufferString(
		`{"bucketKey":"`+bucketKey+`","policyKey":"transient"}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	if resp, err := c.HTTPClient.Do(createReq); err != nil {
		return "", err
	} else if resp.StatusCode != 200 && resp.StatusCode != 201 {
		resp.Body.Close()
		return "", fmt.Errorf("create bucket failed: %d", resp.StatusCode)
	} else {
		resp.Body.Close()
	}

	// Загружаем файл
	uploadURL := fmt.Sprintf("%s/buckets/%s/objects/%s", forgeOSSBase, bucketKey, objectName)
	uploadReq, _ := http.NewRequest("PUT", uploadURL, bytes.NewReader(data))
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed %d: %s", resp.StatusCode, string(b))
	}
	var result struct {
		ObjectID string `json:"objectId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	urn := base64.RawURLEncoding.EncodeToString([]byte(result.ObjectID))
	return urn, nil
}

// StartConversionJob запускает Model Derivative job для конвертации в SVF.
func (c *ForgeClient) StartConversionJob(urn string) error {
	token, err := c.getToken()
	if err != nil {
		return err
	}
	payload := map[string]any{
		"input": map[string]string{"urn": urn},
		"output": map[string]any{
			"formats": []map[string]any{
				{"type": "svf", "views": []string{"2d", "3d"}},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", forgeMDBase+"/designdata/"+urn+"/job", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ads-force", "true")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start job failed %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// WaitForJob ждёт завершения job (polling manifest).
func (c *ForgeClient) WaitForJob(urn string, timeout time.Duration) error {
	token, err := c.getToken()
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", forgeMDBase+"/designdata/"+urn+"/manifest", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		var manifest struct {
			Status   string `json:"status"`
			Progress string `json:"progress"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&manifest)
		resp.Body.Close()
		if manifest.Status == "success" {
			return nil
		}
		if manifest.Status == "failed" || manifest.Status == "timeout" {
			return fmt.Errorf("conversion failed: %s", manifest.Status)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("conversion timeout after %v", timeout)
}
