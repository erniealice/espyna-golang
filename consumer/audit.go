package consumer

import (
	audituc "github.com/erniealice/espyna-golang/internal/application/usecases/service/audit"
)

// Re-export audit use case types for consumer apps that build the
// ListRecentSwitches closure in their composition root.

// ListRecentSwitchesRequest is the re-exported input type for the
// ListRecentSwitches use case.
type ListRecentSwitchesRequest = audituc.ListRecentSwitchesRequest

// ListRecentSwitchesEntry is the re-exported per-row result type.
type ListRecentSwitchesEntry = audituc.ListRecentSwitchesEntry

// ListRecentSwitchesResponse is the re-exported result type.
type ListRecentSwitchesResponse = audituc.ListRecentSwitchesResponse
