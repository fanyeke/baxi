// Package feishu provides HTTP client for Feishu Open API.
package feishu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	feishuBaseURL   = "https://open.feishu.cn/open-apis"
	batchLimit      = 500
	maxRetries      = 3
	defaultWaitBase = 1 * time.Second
)

// BitableClient defines operations on Feishu bitable tables.
type BitableClient interface {
	ListRecords(tableID string, pageSize int, filterConfig map[string]any) ([]map[string]any, error)
	UpsertByKey(tableID string, records []map[string]any, keyField string) (created []map[string]any, updated []map[string]any, err error)
	SendMessage(chatID, content string, dryRun bool) (string, error)
}

// Client communicates with Feishu Open API.
type Client struct {
	appID       string
	appSecret   string
	appToken    string
	dryRun      bool
	baseURL     string
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
}

// NewClient creates a new Feishu API client.
func NewClient(appID, appSecret, appToken string, dryRun bool) *Client {
	return &Client{
		appID:      appID,
		appSecret:  appSecret,
		appToken:   appToken,
		dryRun:     dryRun,
		baseURL:    feishuBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) getTenantAccessToken() (string, error) {
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
	resp, err := c.doRequest("POST", "/auth/v3/tenant_access_token/internal", payload, true)
	if err != nil {
		return "", err
	}

	token, _ := resp["tenant_access_token"].(string)
	expire := 7200
	if e, ok := resp["expire"].(float64); ok {
		expire = int(e)
	}
	c.accessToken = token
	c.tokenExpiry = time.Now().Add(time.Duration(expire-60) * time.Second)
	return token, nil
}

// ListRecords retrieves records from a Feishu bitable table with pagination.
func (c *Client) ListRecords(tableID string, pageSize int, filterConfig map[string]any) ([]map[string]any, error) {
	if c.dryRun {
		return []map[string]any{}, nil
	}

	if pageSize <= 0 || pageSize > 500 {
		pageSize = 500
	}

	var allRecords []map[string]any
	pageToken := ""
	for {
		params := map[string]any{
			"page_size": pageSize,
		}
		if pageToken != "" {
			params["page_token"] = pageToken
		}
		if filterConfig != nil {
			params["filter"] = filterConfig
		}

		resp, err := c.doRequest("GET",
			fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", c.appToken, tableID),
			params, false)
		if err != nil {
			return nil, err
		}

		data, _ := resp["data"].(map[string]any)
		if data == nil {
			break
		}
		items, _ := data["items"].([]any)
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				allRecords = append(allRecords, m)
			}
		}
		hasMore, _ := data["has_more"].(bool)
		if !hasMore {
			break
		}
		pageToken, _ = data["page_token"].(string)
	}
	return allRecords, nil
}

// UpsertByKey upserts records to a Feishu bitable table using a key field for matching.
func (c *Client) UpsertByKey(tableID string, records []map[string]any, keyField string) (created []map[string]any, updated []map[string]any, err error) {
	if c.dryRun {
		for _, r := range records {
			created = append(created, r)
		}
		return created, nil, nil
	}

	existing, err := c.ListRecords(tableID, 500, nil)
	if err != nil {
		return nil, nil, err
	}

	existingMap := make(map[string]map[string]any)
	for _, rec := range existing {
		fields, _ := rec["fields"].(map[string]any)
		if fields == nil {
			continue
		}
		keyVal, _ := fields[keyField]
		if keyVal != nil {
			k := fmt.Sprint(keyVal)
			existingMap[k] = rec
		}
	}

	var toCreate []map[string]any
	for _, record := range records {
		keyVal, _ := record[keyField]
		if keyVal == nil {
			toCreate = append(toCreate, record)
			continue
		}
		k := fmt.Sprint(keyVal)
		if existingRec, ok := existingMap[k]; ok {
			recordID, _ := existingRec["record_id"].(string)
			if recordID != "" {
				updatedRec, err := c.updateRecord(tableID, recordID, record)
				if err == nil && updatedRec != nil {
					updated = append(updated, updatedRec)
				}
			}
		} else {
			toCreate = append(toCreate, record)
		}
	}

	if len(toCreate) > 0 {
		created = c.batchCreate(tableID, toCreate)
	}

	return created, updated, nil
}

func (c *Client) updateRecord(tableID, recordID string, recordData map[string]any) (map[string]any, error) {
	payload := map[string]any{"fields": recordData}
	resp, err := c.doRequest("PUT",
		fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", c.appToken, tableID, recordID),
		payload, false)
	if err != nil {
		return nil, err
	}
	data, _ := resp["data"].(map[string]any)
	return data, nil
}

func (c *Client) batchCreate(tableID string, records []map[string]any) []map[string]any {
	var allCreated []map[string]any
	for i := 0; i < len(records); i += batchLimit {
		end := i + batchLimit
		if end > len(records) {
			end = len(records)
		}
		chunk := records[i:end]
		payload := map[string]any{
			"records": batchToFields(chunk),
		}
		resp, err := c.doRequest("POST",
			fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/batch_create", c.appToken, tableID),
			payload, false)
		if err != nil {
			continue
		}
		data, _ := resp["data"].(map[string]any)
		if data != nil {
			items, _ := data["records"].([]any)
			for _, item := range items {
				if m, ok := item.(map[string]any); ok {
					allCreated = append(allCreated, m)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return allCreated
}

func batchToFields(records []map[string]any) []map[string]any {
	result := make([]map[string]any, len(records))
	for i, r := range records {
		result[i] = map[string]any{"fields": r}
	}
	return result
}

// SendMessage sends a text message to a Feishu chat.
func (c *Client) SendMessage(chatID, content string, dryRun bool) (string, error) {
	isDry := dryRun || c.dryRun
	if isDry {
		return fmt.Sprintf("dry_run_message_%d", time.Now().Unix()), nil
	}

	payload := map[string]any{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    map[string]string{"text": content},
	}
	resp, err := c.doRequest("POST", "/im/v1/messages", payload, false)
	if err != nil {
		return "", err
	}
	data, _ := resp["data"].(map[string]any)
	if data != nil {
		msgID, _ := data["message_id"].(string)
		return msgID, nil
	}
	return "", nil
}

func (c *Client) doRequest(method, path string, body any, skipAuth bool) (map[string]any, error) {
	url := c.baseURL + path

	var token string
	var tokenErr error
	if !skipAuth {
		token, tokenErr = c.getTenantAccessToken()
		if tokenErr != nil {
			return nil, tokenErr
		}
	}

	var bodyJSON []byte
	if body != nil {
		var err error
		bodyJSON, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	waitTimes := []time.Duration{defaultWaitBase, 2 * defaultWaitBase, 4 * defaultWaitBase}
	for attempt, wait := range waitTimes {
		var req *http.Request
		var err error

		switch method {
		case "GET":
			req, err = http.NewRequest(method, url, nil)
			if err != nil {
				return nil, err
			}
			if params, ok := body.(map[string]any); ok {
				q := req.URL.Query()
				for k, v := range params {
					q.Set(k, fmt.Sprint(v))
				}
				req.URL.RawQuery = q.Encode()
			}
		default:
			req, err = http.NewRequest(method, url, strings.NewReader(string(bodyJSON)))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/json")
		}

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < len(waitTimes)-1 {
				time.Sleep(wait)
				continue
			}
			return nil, err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var data map[string]any
		_ = json.Unmarshal(respBody, &data)
		if data == nil {
			data = make(map[string]any)
		}

		code := 0
		if co, ok := data["code"].(float64); ok {
			code = int(co)
		}

		if resp.StatusCode == 429 || code == 170002 {
			if attempt < len(waitTimes)-1 {
				time.Sleep(wait)
				continue
			}
		}

		if code != 0 {
			msg, _ := data["msg"].(string)
			return nil, fmt.Errorf("feishu API error: code=%d, msg=%s", code, msg)
		}

		return data, nil
	}

	return nil, fmt.Errorf("failed %s %s after %d retries", method, path, maxRetries)
}
