// Package rules analyzes and transforms source file bytes.
package rules

// Line is one byte-preserving line of text. Content excludes its line ending;
// Ending contains the exact LF, CRLF, lone CR, or no bytes for an unterminated
// final line. Number is one-based.
//
// The slices refer to the input passed to SplitLines and must be treated as
// read-only.
type Line struct {
	Content []byte
	Ending  []byte
	Number  int
}

// SplitLines divides data at LF, CRLF, and lone CR endings. It does not include
// a phantom empty line after a final ending, and empty input returns no lines.
// Callers that process a UTF-8 BOM must detect and handle it themselves, then
// pass only the bytes after the BOM; line numbering consequently starts at 1
// after the BOM, and Fix remains responsible for preserving or changing it.
func SplitLines(data []byte) []Line {
	if len(data) == 0 {
		return nil
	}

	lines := make([]Line, 0)
	start := 0
	number := 1
	for i := 0; i < len(data); {
		if data[i] != '\n' && data[i] != '\r' {
			i++
			continue
		}

		end := i + 1
		if data[i] == '\r' && end < len(data) && data[end] == '\n' {
			end++
		}
		lines = append(lines, Line{
			Content: data[start:i],
			Ending:  data[i:end],
			Number:  number,
		})
		start = end
		i = end
		number++
	}

	if start < len(data) {
		lines = append(lines, Line{
			Content: data[start:],
			Number:  number,
		})
	}
	return lines
}
