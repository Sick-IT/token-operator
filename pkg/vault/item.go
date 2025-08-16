package vault

// Item defines a common vault item
type Item struct {
	Name  string
	Path  string
	Field string
	Value string
	// TODO: Tags []string
}
