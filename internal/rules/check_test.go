package rules

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/ecfit/internal/config"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		settings map[string]config.Setting
		want     []Finding
	}{
		{name: "empty file has no findings", data: []byte{}, settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: nil},
		{name: "no settings", data: []byte("text\n"), settings: map[string]config.Setting{}, want: []Finding{}},
		{name: "matching LF ending", data: []byte("x\n"), settings: map[string]config.Setting{"end_of_line": {Value: "lf"}}, want: []Finding{}},
		{name: "LF expected but CRLF found", data: []byte("x\r\n"), settings: map[string]config.Setting{"end_of_line": {Value: "lf"}}, want: []Finding{{Line: 1, Code: "EC001", Message: "line ends with crlf, expected lf"}}},
		{name: "CRLF expected but LF found", data: []byte("x\n"), settings: map[string]config.Setting{"end_of_line": {Value: "crlf"}}, want: []Finding{{Line: 1, Code: "EC001", Message: "line ends with lf, expected crlf"}}},
		{name: "matching CRLF ending", data: []byte("x\r\n"), settings: map[string]config.Setting{"end_of_line": {Value: "crlf"}}, want: []Finding{}},
		{name: "matching CR ending", data: []byte("x\r"), settings: map[string]config.Setting{"end_of_line": {Value: "cr"}}, want: []Finding{}},
		{name: "CR expected but LF found", data: []byte("x\n"), settings: map[string]config.Setting{"end_of_line": {Value: "cr"}}, want: []Finding{{Line: 1, Code: "EC001", Message: "line ends with lf, expected cr"}}},
		{name: "unterminated line has no ending diagnostic", data: []byte("x"), settings: map[string]config.Setting{"end_of_line": {Value: "crlf"}}, want: []Finding{}},
		{name: "trailing spaces", data: []byte("x  \n"), settings: map[string]config.Setting{"trim_trailing_whitespace": {Value: "true"}}, want: []Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}}},
		{name: "trailing tab without ending", data: []byte("x\t"), settings: map[string]config.Setting{"trim_trailing_whitespace": {Value: "true"}}, want: []Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}}},
		{name: "trailing whitespace disabled", data: []byte("x \n"), settings: map[string]config.Setting{"trim_trailing_whitespace": {Value: "false"}}, want: []Finding{}},
		{name: "trailing whitespace absent", data: []byte("x\t"), settings: map[string]config.Setting{}, want: []Finding{}},
		{name: "required final newline missing", data: []byte("x"), settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: []Finding{{Line: 1, Code: "EC003", Message: "missing final newline"}}},
		{name: "required final newline uses last line number", data: []byte("x\r\ny"), settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: []Finding{{Line: 2, Code: "EC003", Message: "missing final newline"}}},
		{name: "required final LF is compliant", data: []byte("x\n"), settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: []Finding{}},
		{name: "required final CRLF is compliant", data: []byte("x\r\n"), settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: []Finding{}},
		{name: "required final CR is compliant", data: []byte("x\r"), settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: []Finding{}},
		{name: "final newline forbidden after CR", data: []byte("x\r"), settings: map[string]config.Setting{"insert_final_newline": {Value: "false"}}, want: []Finding{{Line: 1, Code: "EC003", Message: "unexpected final newline"}}},
		{name: "final newline forbidden after LF", data: []byte("x\n"), settings: map[string]config.Setting{"insert_final_newline": {Value: "false"}}, want: []Finding{{Line: 1, Code: "EC003", Message: "unexpected final newline"}}},
		{name: "final newline forbidden after CRLF", data: []byte("x\r\n"), settings: map[string]config.Setting{"insert_final_newline": {Value: "false"}}, want: []Finding{{Line: 1, Code: "EC003", Message: "unexpected final newline"}}},
		{name: "multiple final blank lines use last line number", data: []byte("x\n\n"), settings: map[string]config.Setting{"insert_final_newline": {Value: "false"}}, want: []Finding{{Line: 2, Code: "EC003", Message: "unexpected final newline"}}},
		{name: "final newline forbidden but absent", data: []byte("x"), settings: map[string]config.Setting{"insert_final_newline": {Value: "false"}}, want: []Finding{}},
		{name: "empty file has no final newline finding", data: []byte{}, settings: map[string]config.Setting{"insert_final_newline": {Value: "true"}}, want: nil},
		{name: "space style rejects leading tab", data: []byte("\tx"), settings: map[string]config.Setting{"indent_style": {Value: "space"}}, want: []Finding{{Line: 1, Code: "EC004", Message: "indented with a tab, expected spaces"}}},
		{name: "tab style rejects first leading space", data: []byte(" x"), settings: map[string]config.Setting{"indent_style": {Value: "tab"}}, want: []Finding{{Line: 1, Code: "EC004", Message: "indented with a space, expected tabs"}}},
		{name: "space style rejects tab after spaces", data: []byte(" \tx"), settings: map[string]config.Setting{"indent_style": {Value: "space"}}, want: []Finding{{Line: 1, Code: "EC004", Message: "indented with a tab, expected spaces"}}},
		{name: "blank spaces do not have indentation findings", data: []byte("   \n"), settings: map[string]config.Setting{"indent_style": {Value: "tab"}, "indent_size": {Value: "2"}}, want: []Finding{}},
		{name: "blank tab does not have indentation findings", data: []byte("\t\n"), settings: map[string]config.Setting{"indent_style": {Value: "space"}, "indent_size": {Value: "2"}}, want: []Finding{}},
		{name: "three spaces violate size two", data: []byte("   x"), settings: map[string]config.Setting{"indent_style": {Value: "space"}, "indent_size": {Value: "2"}}, want: []Finding{{Line: 1, Code: "EC005", Message: "indent of 3 spaces is not a multiple of 2"}}},
		{name: "four spaces are a size two multiple", data: []byte("    x"), settings: map[string]config.Setting{"indent_size": {Value: "2"}}, want: []Finding{}},
		{name: "size one accepts two spaces", data: []byte("  x"), settings: map[string]config.Setting{"indent_size": {Value: "1"}}, want: []Finding{}},
		{name: "maximum nine digit size is accepted", data: []byte(" x"), settings: map[string]config.Setting{"indent_size": {Value: "999999999"}}, want: []Finding{{Line: 1, Code: "EC005", Message: "indent of 1 spaces is not a multiple of 999999999"}}},
		{name: "leading zero size is numeric", data: []byte("   x"), settings: map[string]config.Setting{"indent_size": {Value: "000000002"}}, want: []Finding{{Line: 1, Code: "EC005", Message: "indent of 3 spaces is not a multiple of 2"}}},
		{name: "tab indent size is not numeric", data: []byte("   x"), settings: map[string]config.Setting{"indent_size": {Value: "tab"}}, want: []Finding{}},
		{name: "tab style skips numeric indent check", data: []byte("   x"), settings: map[string]config.Setting{"indent_style": {Value: "tab"}, "indent_size": {Value: "2"}}, want: []Finding{{Line: 1, Code: "EC004", Message: "indented with a space, expected tabs"}}},
		{name: "line length at limit", data: []byte("abc"), settings: map[string]config.Setting{"max_line_length": {Value: "3"}}, want: []Finding{}},
		{name: "line length over limit", data: []byte("abcd"), settings: map[string]config.Setting{"max_line_length": {Value: "3"}}, want: []Finding{{Line: 1, Code: "EC006", Message: "line is 4 characters long, the limit is 3"}}},
		{name: "Unicode length counts code points", data: []byte("é🙂"), settings: map[string]config.Setting{"max_line_length": {Value: "1"}}, want: []Finding{{Line: 1, Code: "EC006", Message: "line is 2 characters long, the limit is 1"}}},
		{name: "invalid UTF-8 line is excluded from length", data: []byte{'x', 0xff, 'y'}, settings: map[string]config.Setting{"max_line_length": {Value: "1"}}, want: []Finding{}},
		{name: "UTF-8 forbids a BOM", data: []byte("\xef\xbb\xbfx"), settings: map[string]config.Setting{"charset": {Value: "utf-8"}}, want: []Finding{{Line: 1, Code: "EC007", Message: "unexpected byte order mark"}}},
		{name: "UTF-8 without BOM is compliant", data: []byte("x"), settings: map[string]config.Setting{"charset": {Value: "utf-8"}}, want: []Finding{}},
		{name: "UTF-8 BOM is required", data: []byte("x"), settings: map[string]config.Setting{"charset": {Value: "utf-8-bom"}}, want: []Finding{{Line: 1, Code: "EC007", Message: "missing byte order mark"}}},
		{name: "required UTF-8 BOM is present", data: []byte("\xef\xbb\xbfx"), settings: map[string]config.Setting{"charset": {Value: "utf-8-bom"}}, want: []Finding{}},
		{name: "invalid UTF-8 reported once on first line", data: []byte{0xff, '\n', 0xfe}, settings: map[string]config.Setting{"charset": {Value: "utf-8"}}, want: []Finding{{Line: 1, Code: "EC007", Message: "invalid UTF-8"}}},
		{name: "first invalid UTF-8 line is reported", data: []byte{'o', 'k', '\n', 0xff, '\n', 0xfe}, settings: map[string]config.Setting{"charset": {Value: "utf-8-bom"}}, want: []Finding{{Line: 1, Code: "EC007", Message: "missing byte order mark"}, {Line: 2, Code: "EC007", Message: "invalid UTF-8"}}},
		{name: "invalid UTF-8 is not checked for latin1", data: []byte{0xff}, settings: map[string]config.Setting{"charset": {Value: "latin1"}}, want: []Finding{}},
		{name: "same line findings are ordered by code", data: []byte("\tfoo  \r"), settings: map[string]config.Setting{"end_of_line": {Value: "lf"}, "trim_trailing_whitespace": {Value: "true"}, "insert_final_newline": {Value: "false"}, "indent_style": {Value: "space"}, "indent_size": {Value: "2"}}, want: []Finding{{Line: 1, Code: "EC001", Message: "line ends with cr, expected lf"}, {Line: 1, Code: "EC002", Message: "trailing whitespace"}, {Line: 1, Code: "EC003", Message: "unexpected final newline"}, {Line: 1, Code: "EC004", Message: "indented with a tab, expected spaces"}}},
		{name: "findings are ordered by line then code", data: []byte("\tX  \r\n   Y"), settings: map[string]config.Setting{"end_of_line": {Value: "lf"}, "trim_trailing_whitespace": {Value: "true"}, "insert_final_newline": {Value: "true"}, "indent_style": {Value: "tab"}, "indent_size": {Value: "2"}}, want: []Finding{{Line: 1, Code: "EC001", Message: "line ends with crlf, expected lf"}, {Line: 1, Code: "EC002", Message: "trailing whitespace"}, {Line: 2, Code: "EC003", Message: "missing final newline"}, {Line: 2, Code: "EC004", Message: "indented with a space, expected tabs"}}},
		{name: "BOM does not shift first invalid line", data: []byte("\xef\xbb\xbfok\n\xff\n\xfe"), settings: map[string]config.Setting{"charset": {Value: "utf-8-bom"}}, want: []Finding{{Line: 2, Code: "EC007", Message: "invalid UTF-8"}}},
		{name: "BOM and invalid UTF-8 can both be reported on line one", data: []byte("\xef\xbb\xbf\xff"), settings: map[string]config.Setting{"charset": {Value: "utf-8"}}, want: []Finding{{Line: 1, Code: "EC007", Message: "unexpected byte order mark"}, {Line: 1, Code: "EC007", Message: "invalid UTF-8"}}},
		{name: "off disables maximum line length", data: []byte("long line"), settings: map[string]config.Setting{"max_line_length": {Value: "off"}}, want: []Finding{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Check(test.data, test.settings)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Check(%q, %#v) = %#v, want %#v", test.data, test.settings, got, test.want)
			}
		})
	}
}
