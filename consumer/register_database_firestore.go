//go:build firestore

package consumer

// Activates the Firestore database adapter through the Google contrib module.
// The contrib package's own register_firestore.go keeps the vendor SDK and its
// self-registration behind the same build tag.
import _ "github.com/erniealice/espyna-golang/contrib/google"
