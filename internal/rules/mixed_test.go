package rules

import (
	"reflect"
	"testing"

	"github.com/Gulliby-development/ecfit/internal/config"
)

func TestCheckMixedIndentation(t *testing.T) {
	mixed := []Finding{{Line: 1, Code: "EC008", Message: "indentation mixes tabs and spaces"}}
	tests := []struct {
		name     string
		data     []byte
		settings map[string]config.Setting
		want     []Finding
	}{
		{name: "tab then spaces without a style", data: []byte("\t  x\n"), settings: map[string]config.Setting{}, want: mixed},
		{name: "spaces then tab without a style", data: []byte("  \tx\n"), settings: map[string]config.Setting{}, want: mixed},
		{name: "tabs only", data: []byte("\t\tx\n"), settings: map[string]config.Setting{}, want: []Finding{}},
		{name: "spaces only", data: []byte("    x\n"), settings: map[string]config.Setting{}, want: []Finding{}},
		{name: "space style reports only the tab", data: []byte("  \tx\n"), settings: map[string]config.Setting{"indent_style": {Value: "space"}}, want: []Finding{{Line: 1, Code: "EC004", Message: "indented with a tab, expected spaces"}}},
		{name: "tab style reports only the space", data: []byte(" \tx\n"), settings: map[string]config.Setting{"indent_style": {Value: "tab"}}, want: []Finding{{Line: 1, Code: "EC004", Message: "indented with a space, expected tabs"}}},
		{name: "blank line is skipped", data: []byte("\t \n"), settings: map[string]config.Setting{}, want: []Finding{}},
		{name: "only the leading run counts", data: []byte("x\t y\n"), settings: map[string]config.Setting{}, want: []Finding{}},
		{name: "one finding per line and the right line number", data: []byte("a\n\t \t x\n  b\n \tc\n"), settings: map[string]config.Setting{}, want: []Finding{{Line: 2, Code: "EC008", Message: "indentation mixes tabs and spaces"}, {Line: 4, Code: "EC008", Message: "indentation mixes tabs and spaces"}}},
		{name: "indent size adds no size finding for a mixed run", data: []byte("   \tx\n"), settings: map[string]config.Setting{"indent_size": {Value: "2"}}, want: mixed},
		{name: "same line findings are ordered by code", data: []byte(" \tx \n"), settings: map[string]config.Setting{"trim_trailing_whitespace": {Value: "true"}}, want: []Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}, {Line: 1, Code: "EC008", Message: "indentation mixes tabs and spaces"}}},
		{name: "an unknown style counts as a style", data: []byte(" \tx\n"), settings: map[string]config.Setting{"indent_style": {Value: "mixed"}}, want: []Finding{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Check(test.data, test.settings)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Check() = %#v, want %#v", got, test.want)
			}
		})
	}
}
