package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestData contains all extracted HTTP request data
type RequestData struct {
	Context     context.Context
	PathParams  map[string]string // {id: "123"}
	Headers     map[string]string // {X-Leapfor-MockBusinessType: "education"}
	QueryParams map[string]string // {type: "education", limit: "10"}
	Body        []byte            // Raw request body
}

// GetPathParam gets a path parameter by key
func (r *RequestData) GetPathParam(key string) string {
	if r.PathParams == nil {
		return ""
	}
	return r.PathParams[key]
}

// GetHeader gets a header by key
func (r *RequestData) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	return r.Headers[key]
}

// GetQueryParam gets a query parameter by key
func (r *RequestData) GetQueryParam(key string) string {
	if r.QueryParams == nil {
		return ""
	}
	return r.QueryParams[key]
}

// GetBusinessType gets the business type from headers or query params
func (r *RequestData) GetBusinessType() string {
	// Try header first
	businessType := r.GetHeader("X-Leapfor-MockBusinessType")
	if businessType != "" {
		return businessType
	}

	// Try query param
	businessType = r.GetQueryParam("type")
	if businessType != "" {
		return businessType
	}

	// Default
	return "education"
}

// ReadRequestBody safely reads the request body with limits
func ReadRequestBody(body io.ReadCloser) ([]byte, error) {
	// Limit body size to prevent DoS attacks (10MB limit)
	const maxBodySize = 10 << 20 // 10MB
	limitedReader := io.LimitReader(body, maxBodySize)

	return io.ReadAll(limitedReader)
}

// WriteJSONResponse writes a JSON response to the HTTP response writer
func WriteJSONResponse(w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", "application/json")

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode JSON response: %w", err)
	}

	return nil
}
