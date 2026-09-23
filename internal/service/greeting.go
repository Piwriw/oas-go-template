package service

import (
	"errors"
	"strings"
)

// ErrBlankName identifies a name with no non-whitespace characters.
var ErrBlankName = errors.New("name must not be blank")

// Greeting generates a salutation for a trimmed, nonblank name.
func (s *Service) Greeting(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrBlankName
	}
	return "Hello, " + name + "!", nil
}
