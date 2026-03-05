package handlers

// nullableString converts an empty string to nil so pgx writes SQL NULL instead of an empty string into nullable TEXT columns.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
