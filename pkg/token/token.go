package token

import "time"

type Token struct {
	Name        string
	Description string
	Scopes      []string
	Type        string
	Owner       string
	Value       string
	Expiration  time.Time
}
