package internal

// ContextKey is an empty struct so context keys can be immutable public
// variables with a unique type that nobody else can create.
type ContextKey struct{}
