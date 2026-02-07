//go:build google || firebase

// Package gcp provides shared credential management for Google Cloud Platform services.
//
// This package consolidates common GCP credential handling logic that was previously
// duplicated across google/ and firebase/ packages. It supports multiple authentication
// methods:
//
// 1. Service Account JSON from environment variables
// 2. Service Account JSON file path
// 3. Application Default Credentials (ADC)
//
// Usage example:
//
//	config := gcp.DefaultCredentialConfig("GOOGLE_")
//	opt, err := gcp.GetClientOption(config)
//	if err != nil {
//	    return err
//	}
//	client, err := storage.NewClient(ctx, opt)
//
// The package uses build tag "google" to ensure it's only compiled when
// Google Cloud dependencies are needed, keeping binary sizes small.
package gcp
