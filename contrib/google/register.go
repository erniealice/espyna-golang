// Package google registers Google Cloud and Firebase adapters with espyna's registry.
//
// Import this package with a blank identifier from your application:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/google"
//
// The blank-import alone pulls nothing into the binary. Each adapter family
// (firebase auth, firestore database, gmail email, gcs storage, googlesheets
// tabular) lives in its own register_<adapter>.go file with a matching
// //go:build tag. An adapter's init() fires only when its tag is active —
// so building with -tags firebase pulls only the firebase auth adapter, not
// the unrelated gcs/gmail/firestore/googlesheets code.
//
// This file intentionally has no imports so the package always exists for
// blank-imports even when no Google adapter tag is set.
package google
