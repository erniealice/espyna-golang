// Package microsoft registers the Microsoft Graph email adapter with espyna's registry.
// Blank-import to enable it (registration fires under -tags microsoft_email):
//
//	import _ "github.com/erniealice/espyna-golang/contrib/microsoft"
package microsoft

import _ "github.com/erniealice/espyna-golang/contrib/microsoft/internal/email"
