package schema

// barrel.go is the force-import barrel for the schema registry (Q-DD2-A).
//
// These pb packages have ZERO importers in espyna today — their adapters are
// unwired (the asset domain has only 6 of 10 entities wired; integration_config
// has no adapter; disbursement_method has no adapter — only its FK column is
// referenced as a raw SQL string in treasury/disbursement.go, which does NOT import
// the pb package). Because nothing under contrib/postgres or internal references
// them, their init() never runs and they are absent from
// protoregistry.GlobalTypes — the Build() walk would silently omit them, and the
// boot-shot would then report the live table as "no descriptor" (GAP-C), as
// disbursement_method did on the Plan-2 boot-shot's first real run (2026-05-31).
//
// Each blank import below triggers the pb package's init(), which calls
// protoregistry.GlobalTypes.RegisterMessage, making the message reachable by the
// walk. build.go asserts (assertForceImports) that all resolved tables are present
// after the walk, so dropping an import here fails the boot, not silently.
//
// NOTE: the package names collide (assetv1 x4, integrationv1), but blank (_)
// imports do not bind the package name, so no aliasing is required.
//
// This is a SEPARATE barrel from the adapter-registration barrel at
// contrib/postgres/internal/adapter/imports.go (which blank-imports adapter
// sub-packages and is postgresql-tagged). Keeping the pb force-import here, in the
// dialect-neutral schema package with no build tag, means every dialect
// (mysql/sqlserver validators) gets the same complete GlobalTypes coverage without
// duplicating the import list.
//
// The other ~196 table-annotated pb packages register incidentally via their wired
// adapters' transitive imports; only these need explicit force-import.
//
// See docs/plan/20260530-reflectionless-crud/phase0-findings.md §c (GAP-C).

import (
	_ "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_component"
	_ "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_disposal"
	_ "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_location"
	_ "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_maintenance"
	_ "github.com/erniealice/esqyma/pkg/schema/v1/domain/integration/integration_config"
	_ "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)
