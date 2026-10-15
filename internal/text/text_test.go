package text

import "testing"

func TestQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: `""`},
		{name: "ASCII", input: "Hello, world!", want: `"Hello,?world!"`},
		{name: "printable lower boundary", input: "!", want: `"!"`},
		{name: "printable upper boundary", input: "~", want: `"~"`},
		{name: "space", input: " ", want: `"?"`},
		{name: "control characters", input: "\x00\t\n\x7f", want: `"????"`},
		{name: "Unicode code points", input: "é☃😀", want: `"???"`},
		{name: "mixed ASCII and Unicode", input: "aéB", want: `"a?B"`},
		{name: "backslash remains printable", input: `C:\tmp\file`, want: `"C:\tmp\file"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Quote(test.input); got != test.want {
				t.Errorf("Quote(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestDisplay(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "forward slashes unchanged", input: "src/file.go", want: "src/file.go"},
		{name: "backslashes replaced", input: `C:\work\file.go`, want: "C:/work/file.go"},
		{name: "mixed separators preserved", input: `a\b/c\\d`, want: "a/b/c//d"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Display(test.input); got != test.want {
				t.Errorf("Display(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestPlural(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero is plural", n: 0, want: "files"},
		{name: "one is singular", n: 1, want: "file"},
		{name: "two is plural", n: 2, want: "files"},
		{name: "negative is plural", n: -1, want: "files"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Plural(test.n, "file", "files"); got != test.want {
				t.Errorf("Plural(%d) = %q, want %q", test.n, got, test.want)
			}
		})
	}
}

func TestLowerASCII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "uppercase ASCII", input: "ABC XYZ", want: "abc xyz"},
		{name: "lowercase ASCII unchanged", input: "already lower", want: "already lower"},
		{name: "ASCII bounds", input: "AZaz", want: "azaz"},
		{name: "non-ASCII unchanged", input: "ÄÉΩ", want: "ÄÉΩ"},
		{name: "mixed ASCII and non-ASCII", input: "IÄST", want: "iÄst"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := LowerASCII(test.input); got != test.want {
				t.Errorf("LowerASCII(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
