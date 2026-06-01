//go:build microsoft_email

package consumer

import (
	// Import Microsoft Graph email adapter to trigger registration via init().
	// The adapter is guarded by //go:build (microsoft && microsoftgraph) || microsoft_email.
	//
	// microsoft_email is the SOLE registration tag: this file (//go:build microsoft_email)
	// is the only blank-import of the adapter package anywhere in the tree, so its init()
	// fires only under -tags microsoft_email. That single tag is sufficient to compile AND
	// register the adapter (the email integration route config, the adapter, and the
	// common client all compile under microsoft_email alone).
	//
	// The legacy -tags "microsoft microsoftgraph" set still COMPILES the adapter, but does
	// not register it: nothing blank-imports the adapter package under that tag-set, so its
	// init() never runs. Use microsoft_email to actually wire the Microsoft Graph provider.
	_ "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/email/microsoft"
)
