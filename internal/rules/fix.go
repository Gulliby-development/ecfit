// Package rules analyzes and transforms source file bytes.
package rules

import (
	"bytes"

	"github.com/Gulliby-development/ecfit/internal/config"
)

// Fix applies safe byte-preserving transformations in the specified order.
func Fix(data []byte, settings map[string]config.Setting) []byte {
	// An actually empty file stays empty even when BOM or newline settings apply.
	if len(data) == 0 {
		return data
	}

	const bom = "\xef\xbb\xbf"
	if bytes.Equal(data, []byte(bom)) {
		if settingValue(settings, "charset") == "utf-8" {
			return []byte{}
		}
		return append([]byte(nil), data...)
	}

	body := data
	var bomPrefix []byte
	if bytes.HasPrefix(body, []byte(bom)) {
		bomPrefix = body[:len(bom)]
		body = body[len(bom):]
	}
	switch settingValue(settings, "charset") {
	case "utf-8":
		bomPrefix = nil
	case "utf-8-bom":
		bomPrefix = []byte(bom)
	}

	lines := SplitLines(body)
	if len(lines) == 0 {
		result := make([]byte, 0, len(bomPrefix)+len(body))
		result = append(result, bomPrefix...)
		result = append(result, body...)
		return result
	}

	replacementEnding, hasReplacementEnding := endingFor(settingValue(settings, "end_of_line"))
	trimTrailing := settingValue(settings, "trim_trailing_whitespace") == "true"
	var transformed bytes.Buffer
	transformed.Grow(len(body))
	for _, line := range lines {
		contentEnd := len(line.Content)
		if trimTrailing {
			for contentEnd > 0 && (line.Content[contentEnd-1] == ' ' || line.Content[contentEnd-1] == '\t') {
				contentEnd--
			}
		}
		transformed.Write(line.Content[:contentEnd])
		if len(line.Ending) != 0 {
			if hasReplacementEnding {
				transformed.Write(replacementEnding)
			} else {
				transformed.Write(line.Ending)
			}
		}
	}

	bodyResult := transformed.Bytes()
	insertFinal := settingValue(settings, "insert_final_newline")
	if insertFinal == "true" && len(lines) > 0 && len(lines[len(lines)-1].Ending) == 0 {
		finalEnding := replacementEnding
		if !hasReplacementEnding {
			finalEnding = nil
			for i := len(lines) - 1; i >= 0; i-- {
				if len(lines[i].Ending) != 0 {
					finalEnding = lines[i].Ending
					break
				}
			}
			if finalEnding == nil {
				finalEnding = []byte("\n")
			}
		}
		bodyResult = append(bodyResult, finalEnding...)
	} else if insertFinal == "false" {
		for len(bodyResult) != 0 && (bodyResult[len(bodyResult)-1] == '\r' || bodyResult[len(bodyResult)-1] == '\n') {
			bodyResult = bodyResult[:len(bodyResult)-1]
		}
	}

	result := make([]byte, 0, len(bomPrefix)+len(bodyResult))
	result = append(result, bomPrefix...)
	result = append(result, bodyResult...)
	return result
}
