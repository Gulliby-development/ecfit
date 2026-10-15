package config

import "testing"

func TestValidKnownProperties(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
		want  bool
	}{
		{name: "indent style tab", key: "indent_style", value: "tab", want: true},
		{name: "indent style space", key: "indent_style", value: "space", want: true},
		{name: "indent style tabs", key: "indent_style", value: "tabs", want: false},
		{name: "indent size tab", key: "indent_size", value: "tab", want: true},
		{name: "indent size one", key: "indent_size", value: "1", want: true},
		{name: "indent size leading zeros", key: "indent_size", value: "0002", want: true},
		{name: "indent size maximum digits", key: "indent_size", value: "999999999", want: true},
		{name: "indent size zero", key: "indent_size", value: "0", want: false},
		{name: "indent size all zeros", key: "indent_size", value: "000", want: false},
		{name: "indent size empty", key: "indent_size", value: "", want: false},
		{name: "indent size negative", key: "indent_size", value: "-1", want: false},
		{name: "indent size plus", key: "indent_size", value: "+1", want: false},
		{name: "indent size decimal", key: "indent_size", value: "1.0", want: false},
		{name: "indent size ten digits", key: "indent_size", value: "1234567890", want: false},
		{name: "indent size invalid word", key: "indent_size", value: "space", want: false},
		{name: "tab width one", key: "tab_width", value: "1", want: true},
		{name: "tab width leading zeros", key: "tab_width", value: "0002", want: true},
		{name: "tab width maximum digits", key: "tab_width", value: "999999999", want: true},
		{name: "tab width zero", key: "tab_width", value: "0", want: false},
		{name: "tab width all zeros", key: "tab_width", value: "000", want: false},
		{name: "tab width tab", key: "tab_width", value: "tab", want: false},
		{name: "tab width ten digits", key: "tab_width", value: "1234567890", want: false},
		{name: "line ending lf", key: "end_of_line", value: "lf", want: true},
		{name: "line ending crlf", key: "end_of_line", value: "crlf", want: true},
		{name: "line ending cr", key: "end_of_line", value: "cr", want: true},
		{name: "line ending unix", key: "end_of_line", value: "unix", want: false},
		{name: "charset latin1", key: "charset", value: "latin1", want: true},
		{name: "charset utf-8", key: "charset", value: "utf-8", want: true},
		{name: "charset utf-8 bom", key: "charset", value: "utf-8-bom", want: true},
		{name: "charset utf-16be", key: "charset", value: "utf-16be", want: true},
		{name: "charset utf-16le", key: "charset", value: "utf-16le", want: true},
		{name: "charset ascii", key: "charset", value: "ascii", want: false},
		{name: "trim trailing true", key: "trim_trailing_whitespace", value: "true", want: true},
		{name: "trim trailing false", key: "trim_trailing_whitespace", value: "false", want: true},
		{name: "trim trailing yes", key: "trim_trailing_whitespace", value: "yes", want: false},
		{name: "final newline true", key: "insert_final_newline", value: "true", want: true},
		{name: "final newline false", key: "insert_final_newline", value: "false", want: true},
		{name: "final newline yes", key: "insert_final_newline", value: "yes", want: false},
		{name: "maximum line length off", key: "max_line_length", value: "off", want: true},
		{name: "maximum line length one", key: "max_line_length", value: "1", want: true},
		{name: "maximum line length leading zeros", key: "max_line_length", value: "0002", want: true},
		{name: "maximum line length maximum digits", key: "max_line_length", value: "999999999", want: true},
		{name: "maximum line length zero", key: "max_line_length", value: "0", want: false},
		{name: "maximum line length all zeros", key: "max_line_length", value: "000", want: false},
		{name: "maximum line length empty", key: "max_line_length", value: "", want: false},
		{name: "maximum line length negative", key: "max_line_length", value: "-1", want: false},
		{name: "maximum line length plus", key: "max_line_length", value: "+1", want: false},
		{name: "maximum line length decimal", key: "max_line_length", value: "1.0", want: false},
		{name: "maximum line length ten digits", key: "max_line_length", value: "1234567890", want: false},
		{name: "maximum line length invalid word", key: "max_line_length", value: "none", want: false},
		{name: "unset indent style", key: "indent_style", value: "unset", want: true},
		{name: "unset indent size", key: "indent_size", value: "unset", want: true},
		{name: "unset tab width", key: "tab_width", value: "unset", want: true},
		{name: "unset line ending", key: "end_of_line", value: "unset", want: true},
		{name: "unset charset", key: "charset", value: "unset", want: true},
		{name: "unset trim trailing", key: "trim_trailing_whitespace", value: "unset", want: true},
		{name: "unset final newline", key: "insert_final_newline", value: "unset", want: true},
		{name: "unset maximum line length", key: "max_line_length", value: "unset", want: true},
		{name: "uppercase key and value", key: "INDENT_SIZE", value: "Tab", want: true},
		{name: "uppercase symbolic value", key: "END_OF_LINE", value: "CRLF", want: true},
		{name: "uppercase off", key: "MAX_LINE_LENGTH", value: "OFF", want: true},
		{name: "uppercase boolean", key: "TRIM_TRAILING_WHITESPACE", value: "TRUE", want: true},
		{name: "unknown property is ignored", key: "custom_property", value: "anything", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.key, tc.value); got != tc.want {
				t.Errorf("Valid(%q, %q) = %t, want %t", tc.key, tc.value, got, tc.want)
			}
		})
	}
}
