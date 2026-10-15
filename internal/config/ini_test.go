package config

import (
	"reflect"
	"testing"
)

func TestParseAcceptedInput(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   File
	}{
		{
			name:   "empty input",
			source: "",
			want:   File{Display: "cfg"},
		},
		{
			name:   "blank lines and comments",
			source: " \t\n# comment\n; comment\n",
			want:   File{Display: "cfg"},
		},
		{
			name:   "BOM and CRLF",
			source: "\uFEFFroot = TRUE\r\n\r\n[*]\r\nindent_style = SPACE\r\n",
			want: File{Display: "cfg", Root: true, Sections: []Section{
				{Glob: "*", Props: []Prop{{Key: "indent_style", Value: "space", Line: 4}}},
			}},
		},
		{
			name:   "preamble ignores non-root keys and false root",
			source: "ignored = value\nroot = false\n[*.go]\n",
			want:   File{Display: "cfg", Sections: []Section{{Glob: "*.go"}}},
		},
		{
			name:   "root key and value are ASCII case insensitive",
			source: " RoOt \t= \tTrUe \t",
			want:   File{Display: "cfg", Root: true},
		},
		{
			name:   "section keys and values use ASCII lowercase",
			source: "[*.GO]\n INDENT_STYLE \t= \tTaB \n",
			want: File{Display: "cfg", Sections: []Section{
				{Glob: "*.GO", Props: []Prop{{Key: "indent_style", Value: "tab", Line: 2}}},
			}},
		},
		{
			name:   "unknown property is ignored",
			source: "[*]\nnot_a_property = invalid\nindent_size = 4\n",
			want: File{Display: "cfg", Sections: []Section{
				{Glob: "*", Props: []Prop{{Key: "indent_size", Value: "4", Line: 3}}},
			}},
		},
		{
			name:   "repeated sections and assignments preserve order",
			source: "[*.go]\nindent_size=2\n[*]\nindent_size=4\n[*.go]\nindent_size=8\n",
			want: File{Display: "cfg", Sections: []Section{
				{Glob: "*.go", Props: []Prop{{Key: "indent_size", Value: "2", Line: 2}}},
				{Glob: "*", Props: []Prop{{Key: "indent_size", Value: "4", Line: 4}}},
				{Glob: "*.go", Props: []Prop{{Key: "indent_size", Value: "8", Line: 6}}},
			}},
		},
		{
			name:   "section content is not trimmed",
			source: "[ *.go ]\n",
			want:   File{Display: "cfg", Sections: []Section{{Glob: " *.go "}}},
		},
		{
			name:   "section uses text through the last closing bracket",
			source: "[a] trailing ]\n",
			want:   File{Display: "cfg", Sections: []Section{{Glob: "a] trailing "}}},
		},
		{
			name:   "empty section glob means only an actually empty inner string",
			source: "[ ]\n",
			want:   File{Display: "cfg", Sections: []Section{{Glob: " "}}},
		},
		{
			name:   "CRLF assignment line and source number",
			source: "[*]\r\nindent_size = 2\r\n",
			want:   File{Display: "cfg", Sections: []Section{{Glob: "*", Props: []Prop{{Key: "indent_size", Value: "2", Line: 2}}}}},
		},
		{
			name:   "final unterminated assignment is parsed",
			source: "[*]\ninsert_final_newline = false",
			want: File{Display: "cfg", Sections: []Section{
				{Glob: "*", Props: []Prop{{Key: "insert_final_newline", Value: "false", Line: 2}}},
			}},
		},
		{
			name:   "all recognized property names are retained",
			source: "[*]\nindent_style=space\nindent_size=tab\ntab_width=4\nend_of_line=cr\ncharset=utf-8-bom\ntrim_trailing_whitespace=true\ninsert_final_newline=false\nmax_line_length=off\n",
			want: File{Display: "cfg", Sections: []Section{{
				Glob: "*",
				Props: []Prop{
					{Key: "indent_style", Value: "space", Line: 2},
					{Key: "indent_size", Value: "tab", Line: 3},
					{Key: "tab_width", Value: "4", Line: 4},
					{Key: "end_of_line", Value: "cr", Line: 5},
					{Key: "charset", Value: "utf-8-bom", Line: 6},
					{Key: "trim_trailing_whitespace", Value: "true", Line: 7},
					{Key: "insert_final_newline", Value: "false", Line: 8},
					{Key: "max_line_length", Value: "off", Line: 9},
				},
			}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse("cfg", test.source)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Parse() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "invalid UTF-8",
			source: string([]byte{0xff}),
			want:   "cfg: input is not valid UTF-8",
		},
		{
			name:   "unterminated section header",
			source: "[*.go\n",
			want:   "cfg:1: unterminated section header",
		},
		{
			name:   "a bare opening bracket is unterminated",
			source: "[",
			want:   "cfg:1: unterminated section header",
		},
		{
			name:   "empty section glob",
			source: "[]\n",
			want:   "cfg:1: empty section glob",
		},
		{
			name:   "missing equals",
			source: "not an assignment",
			want:   "cfg:1: expected \"[GLOB]\" or \"KEY = VALUE\"",
		},
		{
			name:   "empty key",
			source: "[*]\n = value",
			want:   "cfg:2: expected \"[GLOB]\" or \"KEY = VALUE\"",
		},
		{
			name:   "split at first equals and preserve remainder in invalid value",
			source: "[*]\nmax_line_length = off=extra\n",
			want:   "cfg:2: invalid value \"off=extra\" for max_line_length",
		},
		{
			name:   "invalid indent style preserves value spelling",
			source: "[*]\nindent_style = Tabs\n",
			want:   "cfg:2: invalid value \"Tabs\" for indent_style",
		},
		{
			name:   "invalid value preserves non-ASCII as one question mark",
			source: "[*]\nindent_style = spáce\n",
			want:   "cfg:2: invalid value \"sp?ce\" for indent_style",
		},
		{
			name:   "invalid value after CRLF has correct line number",
			source: "[*]\r\nindent_size=0\r\n",
			want:   "cfg:2: invalid value \"0\" for indent_size",
		},
		{
			name:   "only first problem is reported",
			source: "[*]\nindent_size=0\nno equals\n",
			want:   "cfg:2: invalid value \"0\" for indent_size",
		},
		{
			name:   "preamble malformed assignment is an error",
			source: " = ignored",
			want:   "cfg:1: expected \"[GLOB]\" or \"KEY = VALUE\"",
		},
		{
			name:   "line beginning with bracket and no closing bracket",
			source: "[a] trailing",
			want:   "cfg:1: unterminated section header",
		},
		{
			name:   "a lone closing bracket is not a section",
			source: "]",
			want:   "cfg:1: expected \"[GLOB]\" or \"KEY = VALUE\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse("cfg", test.source)
			if err == nil {
				t.Fatalf("Parse() error = nil, want %q", test.want)
			}
			if got, want := err.Error(), test.want; got != want {
				t.Errorf("Parse() error = %q, want %q", got, want)
			}
			if !reflect.DeepEqual(got, File{}) {
				t.Errorf("Parse() on error = %#v, want zero File", got)
			}
		})
	}
}
