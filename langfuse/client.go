package langfuse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Langfuse API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string // Langfuse Public Key
	password   string // Langfuse Secret Key
}

// NewClient creates a new Langfuse API client
func NewClient(baseURL, publicKey, secretKey string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		username: publicKey,
		password: secretKey,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

// IngestionEvent represents a single ingestion event
type IngestionEvent struct {
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"`
	Body      map[string]interface{} `json:"body"`
	Metadata  interface{}            `json:"metadata,omitempty"`
}

// IngestionRequest represents the ingestion request
type IngestionRequest struct {
	Batch    []IngestionEvent `json:"batch"`
	Metadata interface{}      `json:"metadata,omitempty"`
}

// IngestionSuccess represents a successful ingestion
type IngestionSuccess struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

// IngestionError represents an ingestion error
type IngestionError struct {
	ID      string      `json:"id"`
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// IngestionResponse represents the ingestion response
type IngestionResponse struct {
	Successes []IngestionSuccess `json:"successes"`
	Errors    []IngestionError   `json:"errors"`
}

// Trace represents a trace object
type Trace struct {
	ID          string      `json:"id"`
	Timestamp   string      `json:"timestamp"`
	Name        string      `json:"name,omitempty"`
	Input       interface{} `json:"input,omitempty"`
	Output      interface{} `json:"output,omitempty"`
	SessionID   string      `json:"sessionId,omitempty"`
	Release     string      `json:"release,omitempty"`
	Version     string      `json:"version,omitempty"`
	UserID      string      `json:"userId,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Public      bool        `json:"public,omitempty"`
	Environment string      `json:"environment,omitempty"`
}

// TraceBody represents the body of a trace event
type TraceBody struct {
	ID          string      `json:"id,omitempty"`
	Timestamp   string      `json:"timestamp,omitempty"`
	Name        string      `json:"name,omitempty"`
	UserID      string      `json:"userId,omitempty"`
	Input       interface{} `json:"input,omitempty"`
	Output      interface{} `json:"output,omitempty"`
	SessionID   string      `json:"sessionId,omitempty"`
	Release     string      `json:"release,omitempty"`
	Version     string      `json:"version,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Environment string      `json:"environment,omitempty"`
	Public      bool        `json:"public,omitempty"`
}

// ObservationLevel represents the observation level
type ObservationLevel string

const (
	ObservationLevelDebug   ObservationLevel = "DEBUG"
	ObservationLevelDefault ObservationLevel = "DEFAULT"
	ObservationLevelWarning ObservationLevel = "WARNING"
	ObservationLevelError   ObservationLevel = "ERROR"
)

// ObservationType represents the observation type
type ObservationType string

const (
	ObservationTypeSpan  ObservationType = "SPAN"
	ObservationTypeEvent ObservationType = "EVENT"
)

// ObservationBody represents the body of an observation
type ObservationBody struct {
	ID                  string                 `json:"id,omitempty"`
	TraceID             string                 `json:"traceId,omitempty"`
	Type                ObservationType        `json:"type"`
	Name                string                 `json:"name,omitempty"`
	StartTime           string                 `json:"startTime,omitempty"`
	EndTime             string                 `json:"endTime,omitempty"`
	CompletionStartTime string                 `json:"completionStartTime,omitempty"`
	Model               string                 `json:"model,omitempty"`
	ModelParameters     map[string]interface{} `json:"modelParameters,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	Version             string                 `json:"version,omitempty"`
	Metadata            interface{}            `json:"metadata,omitempty"`
	Output              interface{}            `json:"output,omitempty"`
	Level               ObservationLevel       `json:"level,omitempty"`
	StatusMessage       string                 `json:"statusMessage,omitempty"`
	ParentObservationID string                 `json:"parentObservationId,omitempty"`
	Environment         string                 `json:"environment,omitempty"`
}

// SDKLogBody represents the body of an SDK log event
type SDKLogBody struct {
	Log interface{} `json:"log"`
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Basic Auth
	req.SetBasicAuth(c.username, c.password)

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// Health checks the health of the API and database
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/public/health", nil)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &healthResp, nil
}

// Ingest sends a batch of tracing events to be ingested
func (c *Client) Ingest(ctx context.Context, req *IngestionRequest) (*IngestionResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/public/ingestion", req)
	if err != nil {
		return nil, fmt.Errorf("ingestion failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle 207 status code (partial success)
	if resp.StatusCode == http.StatusMultiStatus {
		var ingestionResp IngestionResponse
		if err := json.Unmarshal(body, &ingestionResp); err != nil {
			return nil, fmt.Errorf("failed to decode ingestion response: %w", err)
		}
		return &ingestionResp, nil
	}

	// Handle other status codes
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("ingestion failed with status %d: %s", resp.StatusCode, string(body))
	}

	var ingestionResp IngestionResponse
	if err := json.Unmarshal(body, &ingestionResp); err != nil {
		return nil, fmt.Errorf("failed to decode ingestion response: %w", err)
	}

	return &ingestionResp, nil
}

// Helper functions to create different types of events

// CreateTraceEvent creates a trace-create event
func CreateTraceEvent(id, timestamp string, body *TraceBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "trace-create",
		Body:      structToMap(body),
	}
}

// CreateSpanEvent creates a span-create event
func CreateSpanEvent(id, timestamp string, body *ObservationBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "span-create",
		Body:      structToMap(body),
	}
}

// UpdateSpanEvent creates a span-update event
func UpdateSpanEvent(id, timestamp string, body *ObservationBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "span-update",
		Body:      structToMap(body),
	}
}

// CreateEventEvent creates an event-create event
func CreateEventEvent(id, timestamp string, body *ObservationBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "event-create",
		Body:      structToMap(body),
	}
}

// CreateSDKLogEvent creates an sdk-log event
func CreateSDKLogEvent(id, timestamp string, body *SDKLogBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "sdk-log",
		Body:      structToMap(body),
	}
}

// CreateObservationEvent creates an observation-create event
func CreateObservationEvent(id, timestamp string, body *ObservationBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "observation-create",
		Body:      structToMap(body),
	}
}

// UpdateObservationEvent creates an observation-update event
func UpdateObservationEvent(id, timestamp string, body *ObservationBody) *IngestionEvent {
	return &IngestionEvent{
		ID:        id,
		Timestamp: timestamp,
		Type:      "observation-update",
		Body:      structToMap(body),
	}
}

// structToMap converts a struct to a map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	data, _ := json.Marshal(obj)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}
