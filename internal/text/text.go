// Package text provides text formatting helpers shared by ecfit.
package text

import "strings"

// Quote surrounds s with double quotes and replaces every rune outside
// printable ASCII with a question mark.
func Quote(s string) string {
	var quoted strings.Builder
	quoted.Grow(len(s) + 2)
	quoted.WriteByte('"')
	for _, r := range s {
		if r >= '!' && r <= '~' {
			quoted.WriteRune(r)
		} else {
			quoted.WriteByte('?')
		}
	}
	quoted.WriteByte('"')
	return quoted.String()
}

// Display returns a path suitable for display, using forward slashes.
func Display(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// Plural selects the singular form only for a count of one.
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// LowerASCII lowercases ASCII uppercase letters and leaves other bytes intact.
func LowerASCII(s string) string {
	lowered := []byte(s)
	for i, b := range lowered {
		if b >= 'A' && b <= 'Z' {
			lowered[i] = b + ('a' - 'A')
		}
	}
	return string(lowered)
}
