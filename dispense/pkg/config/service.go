package config

// Service handles configuration operations at the service layer
type Service struct{}

// NewService creates a new config service
func NewService() *Service {
	return &Service{}
}

// GetDaytonaAPIKey gets the Daytona API key (interactive or non-interactive)
func (s *Service) GetDaytonaAPIKey(interactive bool) (string, error) {
	if interactive {
		return GetOrPromptAPIKey()
	}
	return GetAPIKeyNonInteractive()
}

// SetDaytonaAPIKey saves the Daytona API key
func (s *Service) SetDaytonaAPIKey(apiKey string) error {
	return SaveAPIKey(apiKey)
}

// ValidateDaytonaAPIKey validates a Daytona API key
func (s *Service) ValidateDaytonaAPIKey(apiKey string) (bool, string) {
	// Basic validation - could be enhanced with actual API call
	if apiKey == "" {
		return false, "API key cannot be empty"
	}

	if len(apiKey) < 10 {
		return false, "API key appears to be too short"
	}

	// In a real implementation, this would make an API call to Daytona to validate the key
	return true, "API key format appears valid"
}