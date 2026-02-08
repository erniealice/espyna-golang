//go:build firebase

package firebase

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/common/gcp"
)

// FirebaseClientManager manages Firebase clients
//
// This replaces the previous singleton pattern with explicit dependency injection,
// making the code more testable and easier to reason about.
type FirebaseClientManager struct {
	app             *firebase.App
	authClient      *auth.Client
	firestoreClient *firestore.Client
	config          *gcp.CredentialConfig
	firestoreDB     string
}

// NewFirebaseClientManager creates a new Firebase client manager
//
// This initializes the Firebase App and prepares it for creating Auth and
// Firestore clients on demand.
func NewFirebaseClientManager(ctx context.Context, firestoreDatabase string) (*FirebaseClientManager, error) {
	// Get credential configuration using shared package
	credConfig := gcp.DefaultCredentialConfig("FIREBASE_")

	// Validate credential config
	if err := credConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid credential config: %w", err)
	}

	// Get Firestore database name (default to environment or "(default)")
	if firestoreDatabase == "" {
		firestoreDatabase = os.Getenv("FIRESTORE_DATABASE")
	}
	if firestoreDatabase == "" {
		firestoreDatabase = "(default)"
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
		app:         app,
		config:      credConfig,
		firestoreDB: firestoreDatabase,
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

// GetFirestoreClient returns or creates the Firestore client
//
// The client is created lazily on first access and cached for subsequent calls.
// It uses the database name provided during manager creation.
func (m *FirebaseClientManager) GetFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	if m.firestoreClient != nil {
		return m.firestoreClient, nil
	}

	var client *firestore.Client
	var err error

	if m.firestoreDB != "" && m.firestoreDB != "(default)" {
		client, err = firestore.NewClientWithDatabase(ctx, m.config.ProjectID, m.firestoreDB)
	} else {
		client, err = firestore.NewClient(ctx, m.config.ProjectID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	m.firestoreClient = client
	log.Printf("✅ Firestore client initialized successfully (database: %s)", m.firestoreDB)
	return m.firestoreClient, nil
}

// GetProjectID returns the Firebase project ID
func (m *FirebaseClientManager) GetProjectID() string {
	return m.config.ProjectID
}

// GetFirestoreDatabase returns the Firestore database name
func (m *FirebaseClientManager) GetFirestoreDatabase() string {
	return m.firestoreDB
}

// Close closes all Firebase clients
func (m *FirebaseClientManager) Close() error {
	if m.firestoreClient != nil {
		return m.firestoreClient.Close()
	}
	return nil
}
