package ai

// ProviderNamer identifies the provider that handles automation Generate calls.
// Circuit-breaker keys use this stable name rather than concrete Go type names.
type ProviderNamer interface {
	ProviderName() string
}
