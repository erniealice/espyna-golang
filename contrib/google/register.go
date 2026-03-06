// Package google registers Google Cloud and Firebase adapters with espyna's registry.
// Import this package with a blank identifier to enable Google/Firebase support:
//
//	import _ "github.com/erniealice/espyna-golang/contrib/google"
package google

import (
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/auth/firebase"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/email/gmail"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/storage/gcs"
	_ "github.com/erniealice/espyna-golang/contrib/google/internal/tabular/googlesheets"
)
