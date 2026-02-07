package registry

// =============================================================================
// Convenience Functions
// =============================================================================

// ListAllAvailableFactories returns a map of provider type to available factory names.
func ListAllAvailableFactories() map[string][]string {
	return map[string][]string{
		"database":    ListAvailableDatabaseProviderFactories(),
		"auth":        ListAvailableAuthProviderFactories(),
		"storage":     ListAvailableStorageProviderFactories(),
		"email":       ListAvailableEmailProviderFactories(),
		"payment":     ListAvailablePaymentProviderFactories(),
		"id":          ListAvailableIDProviderFactories(),
		"translation": ListAvailableTranslationProviderFactories(),
	}
}

// ListAllAvailableBuildFromEnv returns a map of provider type to available BuildFromEnv names.
func ListAllAvailableBuildFromEnv() map[string][]string {
	return map[string][]string{
		"database":    ListAvailableDatabaseBuildFromEnv(),
		"auth":        ListAvailableAuthBuildFromEnv(),
		"storage":     ListAvailableStorageBuildFromEnv(),
		"email":       ListAvailableEmailBuildFromEnv(),
		"payment":     ListAvailablePaymentBuildFromEnv(),
		"id":          ListAvailableIDBuildFromEnv(),
		"translation": ListAvailableTranslationBuildFromEnv(),
	}
}
