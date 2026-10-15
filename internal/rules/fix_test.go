package rules

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Gulliby-development/ecfit/internal/config"
)

func TestFix(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		settings  map[string]config.Setting
		want      []byte
		remaining []Finding
	}{
		{
			name:      "empty stays empty with every byte-ending setting",
			input:     []byte{},
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}, "end_of_line": {Value: "crlf"}, "insert_final_newline": {Value: "true"}, "trim_trailing_whitespace": {Value: "true"}},
			want:      []byte{},
			remaining: nil,
		},
		{
			name:      "BOM-only input is preserved without a newline rule",
			input:     []byte("\xef\xbb\xbf"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}},
			want:      []byte("\xef\xbb\xbf"),
			remaining: []Finding{},
		},
		{
			name:      "BOM-only input does not gain a final newline",
			input:     []byte("\xef\xbb\xbf"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}, "insert_final_newline": {Value: "true"}},
			want:      []byte("\xef\xbb\xbf"),
			remaining: []Finding{},
		},
		{
			name:      "UTF-8 BOM-only input stays empty after charset removal",
			input:     []byte{0xef, 0xbb, 0xbf},
			settings:  map[string]config.Setting{"charset": {Value: "utf-8"}, "insert_final_newline": {Value: "true"}},
			want:      []byte{},
			remaining: nil,
		},
		{
			name:      "UTF-8 removes a BOM from BOM-only input",
			input:     []byte("\xef\xbb\xbf"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8"}},
			want:      []byte{},
			remaining: nil,
		},
		{
			name:      "UTF-8 removes a BOM and preserves text ending",
			input:     []byte("\xef\xbb\xbfx\n"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8"}},
			want:      []byte("x\n"),
			remaining: []Finding{},
		},
		{
			name:      "UTF-8 BOM is added to unterminated text",
			input:     []byte("x"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}},
			want:      []byte("\xef\xbb\xbfx"),
			remaining: []Finding{},
		},
		{
			name:      "existing UTF-8 BOM is not duplicated",
			input:     []byte("\xef\xbb\xbfx\n"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}},
			want:      []byte("\xef\xbb\xbfx\n"),
			remaining: []Finding{},
		},
		{
			name:      "trailing spaces and tabs are removed from every line",
			input:     []byte("x  \n\ty\t"),
			settings:  map[string]config.Setting{"trim_trailing_whitespace": {Value: "true"}},
			want:      []byte("x\n\ty"),
			remaining: []Finding{},
		},
		{
			name:      "trailing whitespace remains when trimming is false",
			input:     []byte("x \t\n"),
			settings:  map[string]config.Setting{"trim_trailing_whitespace": {Value: "false"}},
			want:      []byte("x \t\n"),
			remaining: []Finding{},
		},
		{
			name:      "blank-line spaces and tabs are trimmed",
			input:     []byte(" \t\nx \n"),
			settings:  map[string]config.Setting{"trim_trailing_whitespace": {Value: "true"}},
			want:      []byte("\nx\n"),
			remaining: []Finding{},
		},
		{
			name:      "LF setting replaces CRLF and CR",
			input:     []byte("a\r\nb\rc\n"),
			settings:  map[string]config.Setting{"end_of_line": {Value: "lf"}},
			want:      []byte("a\nb\nc\n"),
			remaining: []Finding{},
		},
		{
			name:      "CRLF setting replaces LF and CR",
			input:     []byte("a\nb\rc"),
			settings:  map[string]config.Setting{"end_of_line": {Value: "crlf"}},
			want:      []byte("a\r\nb\r\nc"),
			remaining: []Finding{},
		},
		{
			name:      "CR setting replaces LF and CRLF",
			input:     []byte("a\r\nb\nc\r"),
			settings:  map[string]config.Setting{"end_of_line": {Value: "cr"}},
			want:      []byte("a\rb\rc\r"),
			remaining: []Finding{},
		},
		{
			name:      "unconfigured line endings stay byte-identical",
			input:     []byte("a\r\nb\rc\n"),
			settings:  map[string]config.Setting{},
			want:      []byte("a\r\nb\rc\n"),
			remaining: []Finding{},
		},
		{
			name:      "configured ending does not add a missing final ending by itself",
			input:     []byte("x"),
			settings:  map[string]config.Setting{"end_of_line": {Value: "crlf"}},
			want:      []byte("x"),
			remaining: []Finding{},
		},
		{
			name:      "required final ending uses configured CRLF",
			input:     []byte("x"),
			settings:  map[string]config.Setting{"end_of_line": {Value: "crlf"}, "insert_final_newline": {Value: "true"}},
			want:      []byte("x\r\n"),
			remaining: []Finding{},
		},
		{
			name:      "required final ending uses the last existing ending",
			input:     []byte("a\nb\r\nd\re"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "true"}},
			want:      []byte("a\nb\r\nd\re\r"),
			remaining: []Finding{},
		},
		{
			name:      "required final ending defaults to LF when no ending exists",
			input:     []byte("x"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "true"}},
			want:      []byte("x\n"),
			remaining: []Finding{},
		},
		{
			name:      "required final ending does not duplicate an existing ending",
			input:     []byte("x\r"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "true"}},
			want:      []byte("x\r"),
			remaining: []Finding{},
		},
		{
			name:      "false removes all contiguous final LF endings",
			input:     []byte("a\n\n\n"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "false"}},
			want:      []byte("a"),
			remaining: []Finding{},
		},
		{
			name:      "false removes mixed contiguous final endings",
			input:     []byte("a\r\nb\n\r"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "false"}},
			want:      []byte("a\r\nb"),
			remaining: []Finding{},
		},
		{
			name:      "false preserves spaces before the removed final ending",
			input:     []byte("a  \n  \n"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "false"}},
			want:      []byte("a  \n  "),
			remaining: []Finding{},
		},
		{
			name:      "false trims whitespace first when trimming is enabled",
			input:     []byte("x \r\n  \r\n"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "false"}, "trim_trailing_whitespace": {Value: "true"}},
			want:      []byte("x"),
			remaining: []Finding{},
		},
		{
			name:      "false can remove the sole line ending",
			input:     []byte("\n"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "false"}},
			want:      []byte{},
			remaining: nil,
		},
		{
			name:      "no final-newline setting preserves unterminated input",
			input:     []byte("last"),
			settings:  map[string]config.Setting{},
			want:      []byte("last"),
			remaining: []Finding{},
		},
		{
			name:      "BOM trimming ending replacement and final insertion follow order",
			input:     []byte("\xef\xbb\xbfa \r\nb\t"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}, "end_of_line": {Value: "crlf"}, "insert_final_newline": {Value: "true"}, "trim_trailing_whitespace": {Value: "true"}},
			want:      []byte("\xef\xbb\xbfa\r\nb\r\n"),
			remaining: []Finding{},
		},
		{
			name:      "invalid bytes survive trimming and line-ending changes",
			input:     []byte{0xff, ' ', '\r'},
			settings:  map[string]config.Setting{"charset": {Value: "utf-8"}, "end_of_line": {Value: "lf"}, "trim_trailing_whitespace": {Value: "true"}},
			want:      []byte{0xff, '\n'},
			remaining: []Finding{{Line: 1, Code: "EC007", Message: "invalid UTF-8"}},
		},
		{
			name:      "invalid bytes survive BOM insertion and final newline repair",
			input:     []byte{0xff, 'x'},
			settings:  map[string]config.Setting{"charset": {Value: "utf-8-bom"}, "insert_final_newline": {Value: "true"}},
			want:      []byte{0xef, 0xbb, 0xbf, 0xff, 'x', '\n'},
			remaining: []Finding{{Line: 1, Code: "EC007", Message: "invalid UTF-8"}},
		},
		{
			name:      "unfixable indentation and line length findings remain",
			input:     []byte(" x  \n"),
			settings:  map[string]config.Setting{"indent_style": {Value: "tab"}, "max_line_length": {Value: "1"}, "trim_trailing_whitespace": {Value: "true"}},
			want:      []byte(" x\n"),
			remaining: []Finding{{Line: 1, Code: "EC004", Message: "indented with a space, expected tabs"}, {Line: 1, Code: "EC006", Message: "line is 2 characters long, the limit is 1"}},
		},
		{
			name:      "line ending changes do not trim whitespace when unset",
			input:     []byte("x  \r"),
			settings:  map[string]config.Setting{"end_of_line": {Value: "lf"}},
			want:      []byte("x  \n"),
			remaining: []Finding{},
		},
		{
			name:      "charset values outside UTF-8 do not alter bytes",
			input:     []byte("\xef\xbb\xbfx\n"),
			settings:  map[string]config.Setting{"charset": {Value: "latin1"}},
			want:      []byte("\xef\xbb\xbfx\n"),
			remaining: []Finding{},
		},
		{
			name:      "CR-only final ending is stripped when false",
			input:     []byte("a\rb\r"),
			settings:  map[string]config.Setting{"insert_final_newline": {Value: "false"}},
			want:      []byte("a\rb"),
			remaining: []Finding{},
		},
		{
			name:      "fixed output is idempotent when unfixable bytes remain",
			input:     []byte("x\xff\t\r"),
			settings:  map[string]config.Setting{"charset": {Value: "utf-8"}, "end_of_line": {Value: "lf"}, "trim_trailing_whitespace": {Value: "true"}},
			want:      []byte{'x', 0xff, '\n'},
			remaining: []Finding{{Line: 1, Code: "EC007", Message: "invalid UTF-8"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Fix(test.input, test.settings)
			if !bytes.Equal(got, test.want) {
				t.Errorf("Fix(%q, %#v) = %q, want %q", test.input, test.settings, got, test.want)
			}
			if findings := Check(got, test.settings); !reflect.DeepEqual(findings, test.remaining) {
				t.Errorf("Check(Fix(...)) = %#v, want %#v", findings, test.remaining)
			}
			if twice := Fix(got, test.settings); !bytes.Equal(twice, got) {
				t.Errorf("Fix is not idempotent: first %q, second %q", got, twice)
			}
		})
	}
}
