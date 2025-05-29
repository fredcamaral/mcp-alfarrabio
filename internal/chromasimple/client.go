package chromasimple

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Simple ChromaDB client without tokenizer dependencies
// This replaces the complex chroma-go with just the HTTP calls we need

type Client interface {
	CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) (Collection, error)
	GetOrCreateCollection(ctx context.Context, name string, metadata map[string]interface{}) (Collection, error)
	Close() error
}

type Collection interface {
	Add(ctx context.Context, ids []string, embeddings [][]float64, documents []string, metadatas []map[string]interface{}) error
	Query(ctx context.Context, queryEmbeddings [][]float64, nResults int, where map[string]interface{}, include []string) (*QueryResult, error)
	Get(ctx context.Context, ids []string, where map[string]interface{}, include []string) (*GetResult, error)
	Delete(ctx context.Context, ids []string) error
	Count(ctx context.Context) (int, error)
}

type QueryResult struct {
	IDs       [][]string                   `json:"ids"`
	Documents [][]string                   `json:"documents"`
	Metadatas [][]map[string]interface{}   `json:"metadatas"`
	Distances [][]float32                  `json:"distances"`
}

type GetResult struct {
	IDs       []string                   `json:"ids"`
	Documents []string                   `json:"documents"`
	Metadatas []map[string]interface{}   `json:"metadatas"`
}

type simpleClient struct {
	baseURL string
	client  *http.Client
}

type simpleCollection struct {
	name   string
	client *simpleClient
}

// NewHTTPClient creates a simple HTTP client for ChromaDB
func NewHTTPClient(baseURL string) Client {
	return &simpleClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *simpleClient) CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) (Collection, error) {
	reqBody := map[string]interface{}{
		"name":     name,
		"metadata": metadata,
	}
	
	_, err := c.makeRequest(ctx, "POST", "/api/v1/collections", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	
	return &simpleCollection{name: name, client: c}, nil
}

func (c *simpleClient) GetOrCreateCollection(ctx context.Context, name string, metadata map[string]interface{}) (Collection, error) {
	reqBody := map[string]interface{}{
		"name":           name,
		"metadata":       metadata,
		"get_or_create":  true,
	}
	
	_, err := c.makeRequest(ctx, "POST", "/api/v1/collections", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}
	
	return &simpleCollection{name: name, client: c}, nil
}

func (c *simpleClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

func (c *simpleCollection) Add(ctx context.Context, ids []string, embeddings [][]float64, documents []string, metadatas []map[string]interface{}) error {
	reqBody := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  documents,
		"metadatas":  metadatas,
	}
	
	path := fmt.Sprintf("/api/v1/collections/%s/add", c.name)
	_, err := c.client.makeRequest(ctx, "POST", path, reqBody)
	return err
}

func (c *simpleCollection) Query(ctx context.Context, queryEmbeddings [][]float64, nResults int, where map[string]interface{}, include []string) (*QueryResult, error) {
	reqBody := map[string]interface{}{
		"query_embeddings": queryEmbeddings,
		"n_results":        nResults,
		"include":          include,
	}
	if where != nil {
		reqBody["where"] = where
	}
	
	path := fmt.Sprintf("/api/v1/collections/%s/query", c.name)
	respBody, err := c.client.makeRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}
	
	var result QueryResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query result: %w", err)
	}
	
	return &result, nil
}

func (c *simpleCollection) Get(ctx context.Context, ids []string, where map[string]interface{}, include []string) (*GetResult, error) {
	reqBody := map[string]interface{}{
		"include": include,
	}
	if len(ids) > 0 {
		reqBody["ids"] = ids
	}
	if where != nil {
		reqBody["where"] = where
	}
	
	path := fmt.Sprintf("/api/v1/collections/%s/get", c.name)
	respBody, err := c.client.makeRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}
	
	var result GetResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get result: %w", err)
	}
	
	return &result, nil
}

func (c *simpleCollection) Delete(ctx context.Context, ids []string) error {
	reqBody := map[string]interface{}{
		"ids": ids,
	}
	
	path := fmt.Sprintf("/api/v1/collections/%s/delete", c.name)
	_, err := c.client.makeRequest(ctx, "POST", path, reqBody)
	return err
}

func (c *simpleCollection) Count(ctx context.Context) (int, error) {
	path := fmt.Sprintf("/api/v1/collections/%s/count", c.name)
	respBody, err := c.client.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return 0, err
	}

	var countResp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(respBody, &countResp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal count result: %w", err)
	}

	return countResp.Count, nil
}

func (c *simpleClient) makeRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.baseURL + path
	
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	return respBody, nil
}