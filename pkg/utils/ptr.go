package utils

// StringPtr returns a pointer to s (useful for optional JSON fields).
func StringPtr(s string) *string {
	return &s
}
