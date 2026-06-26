// Package storage re-exports internal storage helper functions so the separate
// contrib/{aws,azure,google} storage modules can reach them across the
// internal/ Go-module boundary.
//
// Relocated from the former top-level storage/helpers/ as part of the espyna
// public-surface taxonomy (docs/plan/20260610-espyna-public-surface-taxonomy).
// Concern-correct home: shared/storage/ (a sibling of shared/database/), NOT
// shared/database/ — object-id / content-type helpers are a STORAGE concern.
package storage

import internal "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/storage/common"

// GenerateObjectID builds a stable object id from a container/bucket + key.
// Re-exported for the contrib storage adapters (aws/azure/gcs).
var GenerateObjectID = internal.GenerateObjectID
