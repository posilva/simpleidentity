package domain

type ProviderType string

const (
	ProviderTypeGuest  ProviderType = "guest"
	ProviderTypeGoogle ProviderType = "google"
	ProviderTypeApple  ProviderType = "apple"
)
