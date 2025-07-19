package domain

// AuthenticateInput represents the input for the authentication process.
type AuthenticateInput struct {
	ProviderType ProviderType
	AuthData     map[string]string
}

// AuthenticateOutput represents the output of the authentication process.
type AuthenticateOutput struct {
	// AccountID is the unique identifier for the account
	AccountID AccountID
	// IsNew indicates if the account was newly created during authentication
	IsNew bool
}
