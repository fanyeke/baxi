package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	feishuBaseURL   = "https://open.feishu.cn/open-apis"
	maxRetries      = 3
	defaultWaitBase = 1 * time.Second
)

type feishuHTTPClient interface {
	getTenantAccessToken() (string, error)
	sendMessage(chatID, content, msgType string) (string, error)
}

type realFeishuClient struct {
	appID       string
	appSecret   string
	dryRun      bool
	httpClient  *http.Client
	baseURL     string
	accessToken string
	tokenExpiry time.Time
}

func newRealFeishuClient(appID, appSecret string, dryRun bool) *realFeishuClient {
	return &realFeishuClient{
		appID:      appID,
		appSecret:  appSecret,
		dryRun:     dryRun,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    feishuBaseURL,
	}
}

func (c *realFeishuClient) getTenantAccessToken() (string, error) {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}
	if c.dryRun {
		c.accessToken = "dry_run_token"
		c.tokenExpiry = time.Now().Add(2 * time.Hour)
		return c.accessToken, nil
	}

	payload := map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	}
	bodyJSON, _ := json.Marshal(payload)
	url := c.baseURL + "/auth/v3/tenant_access_token/internal"

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var data map[string]any
	_ = json.Unmarshal(respBody, &data)

	token, _ := data["tenant_access_token"].(string)
	expire := 7200
	if e, ok := data["expire"].(float64); ok {
		expire = int(e)
	}
	c.accessToken = token
	c.tokenExpiry = time.Now().Add(time.Duration(expire-60) * time.Second)
	return token, nil
}

func (c *realFeishuClient) sendMessage(chatID, content, msgType string) (string, error) {
	if c.dryRun {
		return fmt.Sprintf("dry_run_message_%d", time.Now().Unix()), nil
	}

	token, err := c.getTenantAccessToken()
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"receive_id": chatID,
		"msg_type":   msgType,
		"content":    map[string]string{"text": content},
	}
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := c.baseURL + "/im/v1/messages?receive_id_type=chat_id"
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	waitTimes := []time.Duration{defaultWaitBase, 2 * defaultWaitBase, 4 * defaultWaitBase}
	for attempt, wait := range waitTimes {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < len(waitTimes)-1 {
				time.Sleep(wait)
				continue
			}
			return "", err
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var data map[string]any
		_ = json.Unmarshal(respBody, &data)
		if data == nil {
			data = make(map[string]any)
		}

		code := 0
		if c, ok := data["code"].(float64); ok {
			code = int(c)
		}

		if resp.StatusCode == 429 || code == 170002 {
			if attempt < len(waitTimes)-1 {
				time.Sleep(wait)
				continue
			}
		}

		if code != 0 {
			msg, _ := data["msg"].(string)
			return "", fmt.Errorf("feishu API error: code=%d, msg=%s", code, msg)
		}

		respData, _ := data["data"].(map[string]any)
		if respData != nil {
			msgID, _ := respData["message_id"].(string)
			return msgID, nil
		}
		return "", nil
	}

	return "", fmt.Errorf("failed to send message after %d retries", maxRetries)
}
