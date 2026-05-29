package dto

// CreateSandboxRequest is the request body for creating a sandbox.
type CreateSandboxRequest struct {
	CaseID string                 `json:"case_id"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// AddProposalToSandboxRequest is the request to add a proposal to a sandbox.
type AddProposalToSandboxRequest struct {
	ProposalID string `json:"proposal_id"`
}

// SandboxResponse is the API response for a sandbox entity.
type SandboxResponse struct {
	SandboxID    string                 `json:"sandbox_id"`
	CaseID       string                 `json:"case_id"`
	ProposalID   *string                `json:"proposal_id,omitempty"`
	Data         map[string]interface{} `json:"data"`
	Status       string                 `json:"status"`
	ComparedWith []string               `json:"compared_with"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    *string                `json:"updated_at,omitempty"`
}

// SandboxListResponse is the paginated list of sandboxes.
type SandboxListResponse struct {
	Items []SandboxResponse `json:"items"`
}

// ComparisonResponse is the API response for a sandbox comparison.
type ComparisonResponse struct {
	Sandbox1ID  string               `json:"sandbox_1_id"`
	Sandbox2ID  string               `json:"sandbox_2_id"`
	Differences []DiffItem           `json:"differences"`
}

// DiffItem represents a single field difference between two sandboxes.
type DiffItem struct {
	Field  string      `json:"field"`
	Value1 interface{} `json:"value_1"`
	Value2 interface{} `json:"value_2"`
}
