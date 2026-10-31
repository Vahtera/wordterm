// Package project defines the project format: metadata, validation and creation.
package project

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

var reserved = map[string]bool{"con": true, "prn": true, "aux": true, "nul": true}

func init() {
	for i := '1'; i <= '9'; i++ {
		reserved["com"+string(i)] = true
		reserved["lpt"+string(i)] = true
	}
}

// ValidateName checks that name is safe as a file or directory name on
// Linux, Windows and macOS.
func ValidateName(n string) error {
	switch {
	case n == "" || strings.TrimSpace(n) != n:
		return fmt.Errorf("name %q must not be empty or start/end with spaces", n)
	case n == "." || n == "..":
		return fmt.Errorf("name %q is not allowed", n)
	case utf8.RuneCountInString(n) > 100:
		return fmt.Errorf("name is longer than 100 characters")
	case strings.HasSuffix(n, "."):
		return fmt.Errorf("name %q must not end with a dot", n)
	}
	for _, r := range n {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return fmt.Errorf("name %q contains the character %q, which is not allowed", n, r)
		}
	}
	if reserved[strings.ToLower(strings.SplitN(n, ".", 2)[0])] {
		return fmt.Errorf("name %q is reserved on Windows", n)
	}
	return nil
}
