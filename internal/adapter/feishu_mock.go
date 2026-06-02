package adapter

// mockFeishuClient is a no-op client for integration tests.
type mockFeishuClient struct{}

func (m *mockFeishuClient) getTenantAccessToken() (string, error) {
	return "mock-token", nil
}

func (m *mockFeishuClient) sendMessage(chatID, content, msgType string) (string, error) {
	return "mock-msg-id", nil
}

// NewMockFeishuClient creates a mock Feishu client for integration tests.
// The returned client implements feishuHTTPClient transparently.
func NewMockFeishuClient() feishuHTTPClient {
	return &mockFeishuClient{}
}
