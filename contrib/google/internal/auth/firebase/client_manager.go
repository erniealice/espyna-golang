package firebase

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/erniealice/espyna-golang/contrib/google/internal/common/gcp"
)

// FirebaseClientManager manages Firebase Auth clients
//
// This replaces the previous singleton pattern with explicit dependency injection,
// making the code more testable and easier to reason about. It is auth-only:
// Firestore lives in the database concern, not here.
type FirebaseClientManager struct {
	app        *firebase.App
	authClient *auth.Client
	config     *gcp.CredentialConfig
}

// NewFirebaseClientManager creates a new Firebase client manager
//
// This initializes the Firebase App and prepares it for creating Auth clients
// on demand.
func NewFirebaseClientManager(ctx context.Context) (*FirebaseClientManager, error) {
	// Get credential configuration using shared package (AUTH/firebase concern).
	credConfig := gcp.DefaultCredentialConfig("AUTH_FIREBASE_")

	// Validate credential config
	if err := credConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credential config: %w", err)
	}

	// Get client option from shared package
	opt, err := gcp.GetClientOption(credConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get client option: %w", err)
	}

	// Create Firebase config
	firebaseConfig := &firebase.Config{
		ProjectID: credConfig.ProjectID,
	}

	// Create Firebase app
	var app *firebase.App
	if opt != nil {
		app, err = firebase.NewApp(ctx, firebaseConfig, opt)
	} else {
		// Use Application Default Credentials
		app, err = firebase.NewApp(ctx, firebaseConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase app: %w", err)
	}

	log.Println("✅ Firebase App initialized successfully")

	return &FirebaseClientManager{
		app:    app,
		config: credConfig,
	}, nil
}

// GetAuthClient returns or creates the Firebase Auth client
//
// The client is created lazily on first access and cached for subsequent calls.
func (m *FirebaseClientManager) GetAuthClient(ctx context.Context) (*auth.Client, error) {
	if m.authClient != nil {
		return m.authClient, nil
	}

	client, err := m.app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	m.authClient = client
	log.Println("✅ Firebase Auth client initialized successfully")
	return m.authClient, nil
}

// GetProjectID returns the Firebase project ID
func (m *FirebaseClientManager) GetProjectID() string {
	return m.config.ProjectID
}

// Close closes all Firebase clients
func (m *FirebaseClientManager) Close() error {
	return nil
}
