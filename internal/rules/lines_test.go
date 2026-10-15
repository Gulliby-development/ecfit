package rules

import (
	"reflect"
	"testing"
)

func testLine(content, ending string, number int) Line {
	var endingBytes []byte
	if ending != "" {
		endingBytes = []byte(ending)
	}
	return Line{
		Content: []byte(content),
		Ending:  endingBytes,
		Number:  number,
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []Line
	}{
		{
			name:  "empty input",
			input: []byte{},
			want:  nil,
		},
		{
			name:  "BOM-only bytes are preserved when passed directly",
			input: []byte("\xef\xbb\xbf"),
			want:  []Line{testLine("\xef\xbb\xbf", "", 1)},
		},
		{
			name:  "one LF ending",
			input: []byte("\n"),
			want:  []Line{testLine("", "\n", 1)},
		},
		{
			name:  "one CR ending",
			input: []byte("\r"),
			want:  []Line{testLine("", "\r", 1)},
		},
		{
			name:  "one CRLF ending",
			input: []byte("\r\n"),
			want:  []Line{testLine("", "\r\n", 1)},
		},
		{
			name:  "unterminated text",
			input: []byte("alpha"),
			want:  []Line{testLine("alpha", "", 1)},
		},
		{
			name:  "terminated LF has no phantom line",
			input: []byte("alpha\n"),
			want:  []Line{testLine("alpha", "\n", 1)},
		},
		{
			name:  "second line is unterminated",
			input: []byte("alpha\nbeta"),
			want:  []Line{testLine("alpha", "\n", 1), testLine("beta", "", 2)},
		},
		{
			name:  "second CRLF line is unterminated",
			input: []byte("alpha\r\nbeta"),
			want:  []Line{testLine("alpha", "\r\n", 1), testLine("beta", "", 2)},
		},
		{
			name:  "lone CR separates lines",
			input: []byte("alpha\rbeta"),
			want:  []Line{testLine("alpha", "\r", 1), testLine("beta", "", 2)},
		},
		{
			name:  "mixed line endings",
			input: []byte("a\nb\r\nc\rd"),
			want: []Line{
				testLine("a", "\n", 1),
				testLine("b", "\r\n", 2),
				testLine("c", "\r", 3),
				testLine("d", "", 4),
			},
		},
		{
			name:  "consecutive LF lines",
			input: []byte("a\n\nb"),
			want: []Line{
				testLine("a", "\n", 1),
				testLine("", "\n", 2),
				testLine("b", "", 3),
			},
		},
		{
			name:  "blank line before text",
			input: []byte("\ntext"),
			want:  []Line{testLine("", "\n", 1), testLine("text", "", 2)},
		},
		{
			name:  "two final LF endings have no phantom third line",
			input: []byte("a\n\n"),
			want:  []Line{testLine("a", "\n", 1), testLine("", "\n", 2)},
		},
		{
			name:  "consecutive lone CR endings",
			input: []byte("a\r\r"),
			want:  []Line{testLine("a", "\r", 1), testLine("", "\r", 2)},
		},
		{
			name:  "CRLF followed by LF",
			input: []byte("a\r\nb\n"),
			want:  []Line{testLine("a", "\r\n", 1), testLine("b", "\n", 2)},
		},
		{
			name:  "invalid bytes are preserved",
			input: []byte("\xffa\xfe\r\n\xfdb"),
			want: []Line{
				testLine("\xffa\xfe", "\r\n", 1),
				testLine("\xfdb", "", 2),
			},
		},
		{
			name:  "UTF-8 content bytes are preserved",
			input: []byte("é🙂\r\n"),
			want:  []Line{testLine("é🙂", "\r\n", 1)},
		},
		{
			name:  "empty CR line before unterminated text",
			input: []byte("a\r\rb"),
			want:  []Line{testLine("a", "\r", 1), testLine("", "\r", 2), testLine("b", "", 3)},
		},
		{
			name:  "all lines terminated without phantom line",
			input: []byte("a\nb\r\nc\r"),
			want: []Line{
				testLine("a", "\n", 1),
				testLine("b", "\r\n", 2),
				testLine("c", "\r", 3),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SplitLines(test.input)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("SplitLines(%q) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}
