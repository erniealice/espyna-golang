package listdata

import (
	"fmt"
	"strconv"
	"strings"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// PaginationUtils provides utilities for handling pagination
type PaginationUtils struct{}

// NewPaginationUtils creates a new pagination utility instance
func NewPaginationUtils() *PaginationUtils {
	return &PaginationUtils{}
}

// ValidatePaginationRequest validates pagination parameters and applies defaults
func (p *PaginationUtils) ValidatePaginationRequest(req *commonpb.PaginationRequest) *commonpb.PaginationRequest {
	if req == nil {
		req = &commonpb.PaginationRequest{
			Limit: 20, // Default limit
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 1},
			},
		}
	}

	// Enforce reasonable limits
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100 // Max limit
	}

	// Ensure valid pagination method
	if req.Method == nil {
		req.Method = &commonpb.PaginationRequest_Offset{
			Offset: &commonpb.OffsetPagination{Page: 1},
		}
	}

	// Validate offset pagination
	if offset, ok := req.Method.(*commonpb.PaginationRequest_Offset); ok {
		if offset.Offset.Page <= 0 {
			offset.Offset.Page = 1
		}
	}

	return req
}

// CreatePaginationResponse creates a pagination response based on request and total count
func (p *PaginationUtils) CreatePaginationResponse(
	req *commonpb.PaginationRequest,
	totalItems int32,
	hasNextData bool,
) *commonpb.PaginationResponse {
	response := &commonpb.PaginationResponse{
		TotalItems: totalItems,
		HasNext:    hasNextData,
	}

	switch method := req.Method.(type) {
	case *commonpb.PaginationRequest_Offset:
		currentPage := method.Offset.Page
		totalPages := (totalItems + req.Limit - 1) / req.Limit // Ceiling division
		if totalPages == 0 {
			totalPages = 1
		}

		response.CurrentPage = &currentPage
		response.TotalPages = &totalPages
		response.HasPrev = currentPage > 1
		response.HasNext = currentPage < totalPages

	case *commonpb.PaginationRequest_Cursor:
		// For cursor pagination, we rely on hasNextData parameter
		// Previous cursor would need to be tracked separately
		response.HasPrev = false // Simplified for now
		if hasNextData {
			nextCursor := p.generateNextCursor(req, totalItems)
			response.NextCursor = &nextCursor
		}
	}

	return response
}

// CalculateOffsetAndLimit calculates SQL offset and limit from pagination request
func (p *PaginationUtils) CalculateOffsetAndLimit(req *commonpb.PaginationRequest) (offset int32, limit int32) {
	limit = req.Limit

	switch method := req.Method.(type) {
	case *commonpb.PaginationRequest_Offset:
		offset = (method.Offset.Page - 1) * req.Limit
	case *commonpb.PaginationRequest_Cursor:
		// For cursor pagination, decode the cursor to get offset
		offset = p.decodeCursor(method.Cursor.Token)
	default:
		offset = 0
	}

	return offset, limit
}

// generateNextCursor creates a cursor token for the next page
func (p *PaginationUtils) generateNextCursor(req *commonpb.PaginationRequest, totalItems int32) string {
	switch method := req.Method.(type) {
	case *commonpb.PaginationRequest_Offset:
		nextOffset := (method.Offset.Page) * req.Limit
		return fmt.Sprintf("offset:%d", nextOffset)
	case *commonpb.PaginationRequest_Cursor:
		currentOffset := p.decodeCursor(method.Cursor.Token)
		nextOffset := currentOffset + req.Limit
		return fmt.Sprintf("offset:%d", nextOffset)
	default:
		return fmt.Sprintf("offset:%d", req.Limit)
	}
}

// decodeCursor extracts offset from cursor token
func (p *PaginationUtils) decodeCursor(token string) int32 {
	if token == "" {
		return 0
	}

	parts := strings.Split(token, ":")
	if len(parts) != 2 || parts[0] != "offset" {
		return 0
	}

	offset, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return 0
	}

	return int32(offset)
}
