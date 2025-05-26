package roots

// Root represents a root directory or URI that the MCP server has access to
type Root struct {
	// URI is the root URI (e.g., "file:///home/user/documents")
	URI string `json:"uri"`
	
	// Name is a human-readable name for the root
	Name string `json:"name"`
	
	// Description provides additional context about the root
	Description string `json:"description,omitempty"`
}

// ListRootsRequest is the request structure for roots/list
type ListRootsRequest struct {
	// No parameters for roots/list
}

// ListRootsResponse is the response structure for roots/list
type ListRootsResponse struct {
	Roots []Root `json:"roots"`
}