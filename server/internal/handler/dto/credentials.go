package dto

// CredentialFieldMetadata reports whether a credential field has a value
// stored server-side, without exposing the value itself.
type CredentialFieldMetadata struct {
	Configured bool `json:"configured"`
}

// CredentialsResponse is the echo shape returned after storing credentials: a
// successful save without needing to re-fetch the whole resource.
type CredentialsResponse struct {
	Fields map[string]CredentialFieldMetadata `json:"fields"`
}
