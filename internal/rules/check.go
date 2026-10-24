// Package rules analyzes and transforms source file bytes.
package rules

import (
	"fmt"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/Gulliby-development/ecfit/internal/config"
)

// Finding is one diagnostic associated with a one-based source line.
type Finding struct {
	Line    int
	Code    string
	Message string
}

// Check reports all enabled diagnostics in stable line and code order.
func Check(data []byte, settings map[string]config.Setting) []Finding {
	if len(data) == 0 {
		return nil
	}

	content := data
	hasBOM := len(content) >= 3 && content[0] == 0xef && content[1] == 0xbb && content[2] == 0xbf
	if hasBOM {
		content = content[3:]
	}

	lines := SplitLines(content)
	findings := make([]Finding, 0)
	charset := settingValue(settings, "charset")
	if charset == "utf-8" && hasBOM {
		findings = append(findings, Finding{Line: 1, Code: "EC007", Message: "unexpected byte order mark"})
	}
	if charset == "utf-8-bom" && !hasBOM {
		findings = append(findings, Finding{Line: 1, Code: "EC007", Message: "missing byte order mark"})
	}

	expectedEnding, hasExpectedEnding := endingFor(settingValue(settings, "end_of_line"))
	trimTrailing := settingValue(settings, "trim_trailing_whitespace") == "true"
	insertFinal := settingValue(settings, "insert_final_newline")
	indentStyle := settingValue(settings, "indent_style")
	indentSize, hasIndentSize := positiveNumber(settingValue(settings, "indent_size"))
	maxLineLength, hasMaxLineLength := positiveNumber(settingValue(settings, "max_line_length"))

	firstInvalidLine := 0
	for _, line := range lines {
		if hasExpectedEnding && len(line.Ending) != 0 && !equalBytes(line.Ending, expectedEnding) {
			findings = append(findings, Finding{
				Line: line.Number, Code: "EC001",
				Message: fmt.Sprintf("line ends with %s, expected %s", endingName(line.Ending), settingValue(settings, "end_of_line")),
			})
		}

		if trimTrailing && hasTrailingWhitespace(line.Content) {
			findings = append(findings, Finding{Line: line.Number, Code: "EC002", Message: "trailing whitespace"})
		}

		leading := leadingIndent(line.Content)
		if !isBlank(line.Content) {
			if indentStyle == "space" && leading.hasTab {
				findings = append(findings, Finding{Line: line.Number, Code: "EC004", Message: "indented with a tab, expected spaces"})
			} else if indentStyle == "tab" && leading.first == ' ' {
				findings = append(findings, Finding{Line: line.Number, Code: "EC004", Message: "indented with a space, expected tabs"})
			}

			if indentStyle == "" && leading.hasTab && leading.hasSpace {
				findings = append(findings, Finding{Line: line.Number, Code: "EC008", Message: "indentation mixes tabs and spaces"})
			}

			if hasIndentSize && indentStyle != "tab" && leading.allSpaces && leading.count%indentSize != 0 {
				findings = append(findings, Finding{
					Line: line.Number, Code: "EC005",
					Message: fmt.Sprintf("indent of %d spaces is not a multiple of %d", leading.count, indentSize),
				})
			}
		}

		if hasMaxLineLength && utf8.Valid(line.Content) {
			length := utf8.RuneCount(line.Content)
			if length > maxLineLength {
				findings = append(findings, Finding{
					Line: line.Number, Code: "EC006",
					Message: fmt.Sprintf("line is %d characters long, the limit is %d", length, maxLineLength),
				})
			}
		}

		if (charset == "utf-8" || charset == "utf-8-bom") && firstInvalidLine == 0 && !utf8.Valid(line.Content) {
			firstInvalidLine = line.Number
		}
	}

	if insertFinal == "true" && len(lines) != 0 && len(lines[len(lines)-1].Ending) == 0 {
		last := lines[len(lines)-1]
		findings = append(findings, Finding{Line: last.Number, Code: "EC003", Message: "missing final newline"})
	} else if insertFinal == "false" && len(lines) != 0 && len(lines[len(lines)-1].Ending) != 0 {
		last := lines[len(lines)-1]
		findings = append(findings, Finding{Line: last.Number, Code: "EC003", Message: "unexpected final newline"})
	}

	if firstInvalidLine != 0 {
		findings = append(findings, Finding{Line: firstInvalidLine, Code: "EC007", Message: "invalid UTF-8"})
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Code < findings[j].Code
	})
	return findings
}

func settingValue(settings map[string]config.Setting, key string) string {
	return settings[key].Value
}

func endingFor(name string) ([]byte, bool) {
	switch name {
	case "lf":
		return []byte("\n"), true
	case "crlf":
		return []byte("\r\n"), true
	case "cr":
		return []byte("\r"), true
	default:
		return nil, false
	}
}

func endingName(ending []byte) string {
	switch {
	case equalBytes(ending, []byte("\r\n")):
		return "crlf"
	case equalBytes(ending, []byte("\n")):
		return "lf"
	case equalBytes(ending, []byte("\r")):
		return "cr"
	default:
		return "unknown"
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func positiveNumber(value string) (int, bool) {
	if len(value) == 0 || len(value) > 9 {
		return 0, false
	}
	for i := range value {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
	}
	number, err := strconv.Atoi(value)
	return number, err == nil && number > 0
}

func hasTrailingWhitespace(content []byte) bool {
	return len(content) != 0 && (content[len(content)-1] == ' ' || content[len(content)-1] == '\t')
}

type indent struct {
	count     int
	first     byte
	hasTab    bool
	hasSpace  bool
	allSpaces bool
}

func leadingIndent(content []byte) indent {
	result := indent{allSpaces: true}
	for _, b := range content {
		if b != ' ' && b != '\t' {
			break
		}
		if result.count == 0 {
			result.first = b
		}
		result.count++
		if b == '\t' {
			result.hasTab = true
			result.allSpaces = false
		} else {
			result.hasSpace = true
		}
	}
	return result
}

func isBlank(content []byte) bool {
	for len(content) != 0 {
		r, size := utf8.DecodeRune(content)
		if r == utf8.RuneError && size == 1 {
			return false
		}
		if !unicode.IsSpace(r) {
			return false
		}
		content = content[size:]
	}
	return true
}
