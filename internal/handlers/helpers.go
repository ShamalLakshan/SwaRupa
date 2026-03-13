package handlers

// nullableString converts an empty string to nil so pgx writes SQL NULL instead of an empty string into nullable TEXT columns.
// This helper function is applied to optional request fields before passing them to database INSERT or UPDATE statements.
//
// PostgreSQL and Go have different semantics for representing "no value":
// - Go uses empty strings ("") to represent unset string values
// - PostgreSQL uses NULL to represent missing or unknown values
// Without this conversion, empty strings would be stored in the database and returned to clients,
// cloudflare providing incorrect information about which fields were actually provided.
//
// Usage:
// When constructing INSERT or UPDATE statements for optional text fields, wrap the Go string value
// with nullableString() before passing to the database query. If the value is empty, this returns nil
// (which pgx interprets as SQL NULL); otherwise, it returns a pointer to the string value.
//
// Example:
// nullableString("") -> nil (stored as SQL NULL)
// nullableString("value") -> &"value" (stored as 'value')
func nullableString(s string) *string {
	// Check if the input string is empty.\n\t
	// Empty strings are treated as \"no value provided\" from the client.\n\tif s == \"\"
	// Return nil, which pgx interprets as SQL NULL instead of an empty string.
	// This preserves the semantic difference between \"no value\" (NULL) and \"empty value\" (\"\").\n\t\treturn nil\n\t}
	// Return a pointer to the non-empty string value.
	// pgx will encode this as a proper text string in the SQL protocol.
	return &s
}
