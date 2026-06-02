package feishu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a feishu client pointed at a test HTTP server.
func newTestClient(ts *httptest.Server) *Client {
	return &Client{
		appID:      "test-app",
		appSecret:  "test-secret",
		appToken:   "test-token",
		dryRun:     false,
		baseURL:    ts.URL,
		httpClient: ts.Client(),
	}
}

// newAuthTestClient creates a client that has an already-cached token,
// so tests don't need to go through getTenantAccessToken first.
func newAuthTestClient(ts *httptest.Server) *Client {
	c := newTestClient(ts)
	c.accessToken = "preauthd-token"
	c.tokenExpiry = time.Now().Add(2 * time.Hour)
	return c
}

// --- doRequest tests -------------------------------------------------------

func TestDoRequest_GET_WithQueryParams(t *testing.T) {
	var receivedPath, receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer preauthd-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": map[string]any{"items": []any{}},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	resp, err := c.doRequest("GET", "/test/v1/resource", map[string]any{
		"page_size": 50,
		"filter":    `{"status":"active"}`,
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "/test/v1/resource", receivedPath)
	assert.Contains(t, receivedQuery, "page_size=50")
	assert.NotNil(t, resp)
}

func TestDoRequest_POST_WithJSONBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer preauthd-token", r.Header.Get("Authorization"))

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "hello", body["text"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	resp, err := c.doRequest("POST", "/im/v1/messages", map[string]any{"text": "hello"}, false)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDoRequest_SkipAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	resp, err := c.doRequest("POST", "/auth/v3/token", map[string]string{"k": "v"}, true)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDoRequest_APIErrorCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 99999,
			"msg":  "test error",
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.doRequest("GET", "/test", nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feishu API error: code=99999")
}

func TestDoRequest_RetryOn429(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{"code": 170002, "msg": "rate limit"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	resp, err := c.doRequest("GET", "/test", nil, false)
	require.NoError(t, err)
	assert.Equal(t, 3, attempt) // first 2 fail, 3rd succeeds
	assert.NotNil(t, resp)
}

func TestDoRequest_ExhaustRetries(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"code": 170002, "msg": "rate limit"})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.doRequest("GET", "/test", nil, false)
	require.Error(t, err)
	assert.Equal(t, 3, attempt)
	assert.Contains(t, err.Error(), "feishu API error")
}

func TestDoRequest_ConnectionError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.doRequest("GET", "/test", nil, false)
	require.Error(t, err)
}

// --- getTenantAccessToken tests --------------------------------------------

func TestGetTenantAccessToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/v3/tenant_access_token/internal", r.URL.Path)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "test-app", body["app_id"])
		assert.Equal(t, "test-secret", body["app_secret"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"tenant_access_token": "live-token",
			"expire":              7200,
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	token, err := c.getTenantAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "live-token", token)
	assert.Equal(t, "live-token", c.accessToken)
}

func TestGetTenantAccessToken_Cached(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"tenant_access_token": "live-token",
			"expire":              7200,
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)

	token1, err := c.getTenantAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "live-token", token1)

	token2, err := c.getTenantAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "live-token", token2)

	assert.Equal(t, 1, callCount)
}

func TestGetTenantAccessToken_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 10003,
			"msg":  "invalid app id",
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.getTenantAccessToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feishu API error")
}

// --- ListRecords integration tests -----------------------------------------

func TestListRecords_SinglePage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/bitable/v1/apps/test-token/tables/tb-1/records")
		assert.Equal(t, "page_size=100", r.URL.RawQuery)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": map[string]any{
				"items": []any{
					map[string]any{"record_id": "rec-1", "fields": map[string]any{"name": "alice"}},
					map[string]any{"record_id": "rec-2", "fields": map[string]any{"name": "bob"}},
				},
				"has_more": false,
			},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records, err := c.ListRecords("tb-1", 100, nil)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "rec-1", records[0]["record_id"])
}

func TestListRecords_Pagination(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")

		if calls == 1 {
			assert.NotContains(t, r.URL.RawQuery, "page_token")
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []any{
						map[string]any{"record_id": "rec-1"},
					},
					"has_more":  true,
					"page_token": "next-page",
				},
			})
		} else {
			assert.Contains(t, r.URL.RawQuery, "page_token=next-page")
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []any{
						map[string]any{"record_id": "rec-2"},
					},
					"has_more": false,
				},
			})
		}
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records, err := c.ListRecords("tb-1", 50, nil)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, 2, calls)
}

func TestListRecords_WithFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "filter")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"items":    []any{},
				"has_more": false,
			},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records, err := c.ListRecords("tb-1", 50, map[string]any{"field": "value"})
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestListRecords_EmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": nil})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records, err := c.ListRecords("tb-1", 100, nil)
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestListRecords_ClampsPageSize(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "page_size=500")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"items": []any{}, "has_more": false},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.ListRecords("tb-1", 1000, nil)
	require.NoError(t, err)
}

func TestListRecords_ErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 99999,
			"msg":  "list error",
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.ListRecords("tb-1", 100, nil)
	require.Error(t, err)
}

// --- SendMessage integration tests -----------------------------------------

func TestSendMessage_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/im/v1/messages", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "chat-1", body["receive_id"])
		assert.Equal(t, "text", body["msg_type"])
		content, _ := body["content"].(map[string]any)
		assert.Equal(t, "hello", content["text"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": map[string]any{
				"message_id": "msg-12345",
			},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	msgID, err := c.SendMessage("chat-1", "hello", false)
	require.NoError(t, err)
	assert.Equal(t, "msg-12345", msgID)
}

func TestSendMessage_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 99999,
			"msg":  "send error",
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.SendMessage("chat-1", "hello", false)
	require.Error(t, err)
}

func TestSendMessage_NilResponseData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": nil,
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	msgID, err := c.SendMessage("chat-1", "hello", false)
	require.NoError(t, err)
	assert.Equal(t, "", msgID)
}

// --- UpsertByKey integration tests -----------------------------------------

func TestUpsertByKey_CreateOnly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && r.URL.Path == "/bitable/v1/apps/test-token/tables/tb-1/records":
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"items": []any{}, "has_more": false},
			})

		case r.Method == "POST" && r.URL.Path == "/bitable/v1/apps/test-token/tables/tb-1/records/batch_create":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			records, _ := body["records"].([]any)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"records": records,
				},
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records := []map[string]any{
		{"name": "alice", "role": "admin"},
		{"name": "bob", "role": "user"},
	}

	created, updated, err := c.UpsertByKey("tb-1", records, "name")
	require.NoError(t, err)
	require.Len(t, created, 2)
	require.Len(t, updated, 0)
}

func TestUpsertByKey_UpdateExisting(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && r.URL.Path == "/bitable/v1/apps/test-token/tables/tb-1/records":
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"items": []any{
						map[string]any{
							"record_id": "rec-existing-1",
							"fields":    map[string]any{"name": "alice", "role": "old"},
						},
					},
					"has_more": false,
				},
			})

		case r.Method == "PUT" && r.URL.Path == "/bitable/v1/apps/test-token/tables/tb-1/records/rec-existing-1":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, map[string]any{"fields": map[string]any{"name": "alice", "role": "admin"}}, body)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"record_id": "rec-existing-1",
				},
			})

		case r.Method == "POST" && r.URL.Path == "/bitable/v1/apps/test-token/tables/tb-1/records/batch_create":
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"records": []any{
						map[string]any{"record_id": "rec-new-1"},
					},
				},
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records := []map[string]any{
		{"name": "alice", "role": "admin"},   // existing → update
		{"name": "charlie", "role": "viewer"}, // new → create
	}

	created, updated, err := c.UpsertByKey("tb-1", records, "name")
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Len(t, updated, 1)
}

func TestUpsertByKey_MissingKeyField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"items": []any{}, "has_more": false},
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"records": []any{map[string]any{"record_id": "rec-new"}},
			},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	records := []map[string]any{
		{"role": "admin"},
	}

	created, updated, err := c.UpsertByKey("tb-1", records, "name")
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Len(t, updated, 0)
}

// --- updateRecord tests ----------------------------------------------------

func TestUpdateRecord_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"record_id": "rec-1"},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	resp, err := c.updateRecord("tb-1", "rec-1", map[string]any{"name": "alice"})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestUpdateRecord_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 99999,
			"msg":  "update error",
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	_, err := c.updateRecord("tb-1", "rec-1", map[string]any{"name": "alice"})
	require.Error(t, err)
}

// --- batchCreate tests -----------------------------------------------------

func TestBatchCreate_SingleChunk(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		records, _ := body["records"].([]any)
		assert.Len(t, records, 2)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"records": []any{
					map[string]any{"record_id": "rec-1"},
					map[string]any{"record_id": "rec-2"},
				},
			},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	result := c.batchCreate("tb-1", []map[string]any{
		{"name": "a"},
		{"name": "b"},
	})
	assert.Len(t, result, 2)
}

func TestBatchCreate_MultipleChunks(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		records, _ := body["records"].([]any)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"records": records,
			},
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	largeSet := make([]map[string]any, 1100)
	for i := range largeSet {
		largeSet[i] = map[string]any{"idx": i}
	}

	result := c.batchCreate("tb-1", largeSet)
	assert.Len(t, result, 1100)
	assert.Equal(t, 3, callCount)
}

func TestBatchCreate_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 99999,
			"msg":  "batch error",
		})
	}))
	defer ts.Close()

	c := newAuthTestClient(ts)
	result := c.batchCreate("tb-1", []map[string]any{{"name": "a"}})
	assert.Empty(t, result)
}
