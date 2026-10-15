// Package config parses EditorConfig files and validates their supported properties.
package config

import "github.com/Gulliby-development/ecfit/internal/text"

// Valid reports whether value is permitted for key. Unknown properties are
// accepted here because EditorConfig permits clients to ignore unsupported keys.
func Valid(key, value string) bool {
	key = text.LowerASCII(key)
	value = text.LowerASCII(value)

	if value == "unset" {
		return true
	}

	switch key {
	case "indent_style":
		return value == "tab" || value == "space"
	case "indent_size":
		return value == "tab" || positiveInteger(value)
	case "tab_width":
		return positiveInteger(value)
	case "end_of_line":
		return value == "lf" || value == "crlf" || value == "cr"
	case "charset":
		return value == "latin1" || value == "utf-8" || value == "utf-8-bom" ||
			value == "utf-16be" || value == "utf-16le"
	case "trim_trailing_whitespace", "insert_final_newline":
		return value == "true" || value == "false"
	case "max_line_length":
		return value == "off" || positiveInteger(value)
	default:
		return true
	}
}

func knownProperty(key string) bool {
	switch key {
	case "indent_style", "indent_size", "tab_width", "end_of_line", "charset",
		"trim_trailing_whitespace", "insert_final_newline", "max_line_length":
		return true
	default:
		return false
	}
}

func positiveInteger(value string) bool {
	if len(value) == 0 || len(value) > 9 {
		return false
	}

	positive := false
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
		if value[i] != '0' {
			positive = true
		}
	}
	return positive
}
