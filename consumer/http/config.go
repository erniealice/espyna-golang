package http

// AppConfig is the application-level configuration the consumer app threads into
// the fluent Server via WithApp (A.1 #9). It is read from env in the app's
// config.go (appConfig()) and carries NO domain imports — pure scalars + the
// Features flags. The Server uses it to fill the chain's BusinessType, the
// reserved workspace slugs, the asset root, and the feature gates.
type AppConfig struct {
	// ID is the application identifier (e.g. "service-admin").
	ID string

	// Name is the human-facing application name.
	Name string

	// DefaultTheme is the pyeza theme preset (e.g. "corporate-steel").
	DefaultTheme string

	// DefaultFont is the default font key (e.g. "default").
	DefaultFont string

	// DefaultBusinessType is the business-type tier driving the lyngua label
	// cascade (e.g. "general", "professional", "service").
	DefaultBusinessType string

	// AssetRoot is the directory served at /assets/ (defaults to "assets").
	AssetRoot string

	// ReservedWorkspaceSlugs are workspace slug values that cannot be claimed as
	// workspace slugs (e.g. "auth", "me", "portal").
	ReservedWorkspaceSlugs []string

	// Features carries the compile-time feature gates.
	Features Features
}

// Features carries application feature gates threaded through AppConfig.
type Features struct {
	// SupplierPortalReady gates the SUPPLIER portal persona. DEFAULT false =
	// supplier ships dark. Enabling it in shadow mode stays a fatal inside espyna
	// (it requires AUTHZ_ENFORCE truthy).
	SupplierPortalReady bool
}
