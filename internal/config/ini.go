package config

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Gulliby-development/ecfit/internal/text"
)

// Parse parses EditorConfig text, retaining recognized properties in source
// order and attaching each property to its source line.
func Parse(display, source string) (File, error) {
	file := File{Display: display}
	if !utf8.ValidString(source) {
		return File{}, fmt.Errorf("%s: input is not valid UTF-8", display)
	}

	source = strings.TrimPrefix(source, "\uFEFF")
	var section *Section
	for lineIndex, rawLine := range configLines(source) {
		lineNumber := lineIndex + 1
		line := strings.Trim(rawLine, " \t")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return File{}, fmt.Errorf("%s:%d: unterminated section header", display, lineNumber)
			}
			open := strings.IndexByte(line, '[')
			close := strings.LastIndexByte(line, ']')
			glob := line[open+1 : close]
			if glob == "" {
				return File{}, fmt.Errorf("%s:%d: empty section glob", display, lineNumber)
			}
			file.Sections = append(file.Sections, Section{Glob: glob})
			section = &file.Sections[len(file.Sections)-1]
			continue
		}

		equal := strings.IndexByte(line, '=')
		if equal < 0 {
			return File{}, fmt.Errorf("%s:%d: expected \"[GLOB]\" or \"KEY = VALUE\"", display, lineNumber)
		}
		key := strings.Trim(line[:equal], " \t")
		value := strings.Trim(line[equal+1:], " \t")
		if key == "" {
			return File{}, fmt.Errorf("%s:%d: expected \"[GLOB]\" or \"KEY = VALUE\"", display, lineNumber)
		}

		lowerKey := text.LowerASCII(key)
		lowerValue := text.LowerASCII(value)
		if section == nil {
			if lowerKey == "root" && lowerValue == "true" {
				file.Root = true
			}
			continue
		}
		if !knownProperty(lowerKey) {
			continue
		}
		if !Valid(lowerKey, lowerValue) {
			return File{}, fmt.Errorf("%s:%d: invalid value %s for %s",
				display, lineNumber, text.Quote(value), lowerKey)
		}
		section.Props = append(section.Props, Prop{
			Key:   lowerKey,
			Value: lowerValue,
			Line:  lineNumber,
		})
	}
	return file, nil
}

func configLines(source string) []string {
	if source == "" {
		return []string{""}
	}

	lines := make([]string, 0, strings.Count(source, "\n")+1)
	start := 0
	for i := 0; i < len(source); i++ {
		if source[i] != '\n' {
			continue
		}
		end := i
		if end > start && source[end-1] == '\r' {
			end--
		}
		lines = append(lines, source[start:end])
		start = i + 1
	}
	if start < len(source) {
		lines = append(lines, source[start:])
	}
	return lines
}
