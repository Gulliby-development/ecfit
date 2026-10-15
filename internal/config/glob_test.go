package config

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		glob string
		path string
		want bool
	}{
		// Pinned basename and path matching examples.
		{"*.go", "main.go", true},
		{"*.go", "src/a/main.go", true},
		{"*.go", "main.gox", false},
		{"/main.go", "main.go", true},
		{"/main.go", "src/main.go", false},
		{"src/*.go", "src/a.go", true},
		{"src/*.go", "src/x/a.go", false},
		{"src/*.go", "x/src/a.go", false},
		{"src/**.go", "src/a.go", true},
		{"src/**.go", "src/x/y/a.go", true},
		{"src/**/a.go", "src/a.go", true},
		{"src/**/a.go", "src/x/y/a.go", true},
		{"src/**/a.go", "srcx/a.go", false},
		{"?.txt", "a.txt", true},
		{"?.txt", "ab.txt", false},
		{"a?b", "a/b", false},
		{"[a-c].c", "b.c", true},
		{"[a-c].c", "d.c", false},
		{"[!abc].c", "b.c", false},
		{"[!abc].c", "d.c", true},
		{"*.{js,ts}", "dir/x.ts", true},
		{"{a,b}/*.c", "b/x.c", true},
		{"{a,b}/*.c", "c/x.c", false},
		{"x{a,{b,c}}y", "xcy", true},
		{"file{1..3}.txt", "file2.txt", true},
		{"file{1..3}.txt", "file4.txt", false},
		{"file{1..3}.txt", "file02.txt", false},
		{"file{3..1}.txt", "file2.txt", true},
		{"file{-1..1}.txt", "file-1.txt", true},
		{"{one}.txt", "{one}.txt", true},
		{"{one}.txt", "one.txt", false},
		{"a[", "a[", true},
		{"{a,b", "{a,b", true},
		{`\*.txt`, "*.txt", true},
		{`\*.txt`, "a.txt", false},
		{"*", "a/b/c", true},
		{"a/**", "a/b/c", true},
		{"**", "a/b", true},

		// Stars, directory prefixes, and slash boundaries.
		{"*", "", true},
		{"a*", "a", true},
		{"a*", "ab", true},
		{"a*", "a/b", false},
		{"a/**", "a/", true},
		{"a/**", "a", false},
		{"**/file.txt", "file.txt", true},
		{"**/file.txt", "x/y/file.txt", true},
		{"**/", "a/b", false},
		{"a**b", "a/x/y/b", true},
		{"a*b", "a/x/y/b", false},
		{"[ab].txt", "dir/a.txt", true},
		{"src/**/a.go", "src//a.go", false},

		// Question marks and character classes operate on Unicode code points.
		{"?.txt", "é.txt", true},
		{"??", "éx", true},
		{"?", "", false},
		{"?", "/", false},
		{"[a-c]", "/", false},
		{"[!a]", "/", false},
		{"[é-ê].txt", "ê.txt", true},
		{"[é-ê].txt", "ë.txt", false},
		{`[\]]`, "]", true},
		{"[]", "[]", true},
		{"[abc", "[abc", true},
		{"[abc", "abc", false},
		{"[!a-c]", "z", true},
		{"[!a-c]", "b", false},

		// Alternatives can be empty, nested, and contain glob syntax.
		{"{,a}", "", true},
		{"{,a}", "a", true},
		{"a{,b}c", "ac", true},
		{"a{,b}c", "abc", true},
		{"{a,{b,c}}", "b", true},
		{"{a,{b,c}}", "c", true},
		{"{a,{b,c}}", "d", false},
		{"{src,lib}/?.go", "lib/x.go", true},
		{"{src,lib}/?.go", "other/x.go", false},
		{"{a,b}", "a", true},
		{"{a,b}", "b", true},
		{"{a,b}", "ab", false},
		{"{}", "{}", true},
		{"{one}", "{one}", true},
		{"{a,b", "{a,b", true},

		// Numeric ranges are inclusive, signed, bounded, and canonical in paths.
		{"file{1..3}.txt", "file1.txt", true},
		{"file{1..3}.txt", "file3.txt", true},
		{"file{1..3}.txt", "file0.txt", false},
		{"file{-1..1}.txt", "file0.txt", true},
		{"file{-1..1}.txt", "file1.txt", true},
		{"file{-1..1}.txt", "file-0.txt", false},
		{"n{-999999999..999999999}", "n-999999999", true},
		{"n{-999999999..999999999}", "n999999999", true},
		{"n{999999999..999999998}", "n999999998", true},
		{"n{1..3}", "n0002", false},
		{"n{1..3}", "n+2", false},
		{"n{1000000000..1000000001}", "n1000000000", false},
		{"n{1000000000..1000000001}", "n{1000000000..1000000001}", true},
		{"n{+1..3}", "n{+1..3}", true},
		{"n{+1..3}", "n2", false},
		{"x{1..3}{a,b}", "x2b", true},
		{"x{1..3}{a,b}", "x4b", false},

		// Escapes quote the next character; matching remains case-sensitive.
		{`\?`, "?", true},
		{`\[`, "[", true},
		{`\{a,b\}`, "{a,b}", true},
		{`a\/b`, "a/b", true},
		{`a\/b`, "x/a/b", false},
		{`\`, `\`, true},
		{"Readme.md", "readme.md", false},
		{"A/*.TXT", "A/x.TXT", true},
		{"A/*.TXT", "a/x.TXT", false},
		{"//main.go", "main.go", false},
	}

	for _, test := range tests {
		if got := Match(test.glob, test.path); got != test.want {
			t.Errorf("Match(%q, %q) = %t, want %t", test.glob, test.path, got, test.want)
		}
	}
}
