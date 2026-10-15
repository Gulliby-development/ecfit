package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestREADMEContract(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	document := string(readme)

	for _, heading := range []string{
		"## Install",
		"## Usage",
		"## Settings",
		"## Rules",
		"## Globs",
		"## Exit codes",
		"## Development",
	} {
		if !strings.Contains(document, heading+"\n") {
			t.Errorf("README is missing heading %q", heading)
		}
	}

	for _, required := range []string{
		"go build -o ecfit .",
		"go run . ARGS",
		"go vet ./...",
		"gofmt -l .",
		"go test ./...",
	} {
		if !strings.Contains(document, required) {
			t.Errorf("README is missing %q", required)
		}
	}

	for _, block := range []string{
		"usage: ecfit explain [--root DIR] FILE\n       ecfit check [--root DIR] PATH...\n       ecfit fix [--root DIR] PATH...",
		"root = true\n\n[*]\nend_of_line = lf\ninsert_final_newline = true\ntrim_trailing_whitespace = true\ncharset = utf-8\n\n[*.{go,mod}]\nindent_style = tab\n\n[*.{js,json,yml}]\nindent_style = space\nindent_size = 2\n\n[Makefile]\nindent_style = tab\n\n[docs/**.md]\nmax_line_length = 80\ntrim_trailing_whitespace = false",
		"web/app.js\n  charset = utf-8                  .editorconfig:7\n  end_of_line = lf                 .editorconfig:4\n  indent_size = 4                  web/.editorconfig:2\n  indent_style = space             .editorconfig:13\n  insert_final_newline = true      .editorconfig:5\n  max_line_length = 60             web/.editorconfig:3\n  trim_trailing_whitespace = true  .editorconfig:6",
		"./data.json:1: EC007 unexpected byte order mark\n./docs/guide.md:4: EC001 line ends with crlf, expected lf\n./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80\n./main.go:4: EC002 trailing whitespace\n./util.go:4: EC004 indented with a space, expected tabs\n./util.go:5: EC003 missing final newline\n./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4\n./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4\n./web/app.js:4: EC004 indented with a tab, expected spaces\n./web/app.js:5: EC006 line is 63 characters long, the limit is 60\n10 problems in 5 files, 8 files checked",
		"./data.json: fixed EC007\n./docs/guide.md: fixed EC001\n./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80\n./main.go: fixed EC002\n./util.go: fixed EC003\n./util.go:4: EC004 indented with a space, expected tabs\n./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4\n./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4\n./web/app.js:4: EC004 indented with a tab, expected spaces\n./web/app.js:5: EC006 line is 63 characters long, the limit is 60\nfixed 4 files, 6 problems left",
	} {
		if !strings.Contains(document, block) {
			t.Errorf("README is missing or changed a required example block:\n%s", block)
		}
	}

	if count := strings.Count(document, "```text\n"); count != 4 {
		t.Errorf("README has %d text example blocks, want exactly 4", count)
	}
}

func TestMITLicense(t *testing.T) {
	license, err := os.ReadFile(filepath.Join("..", "..", "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	const want = `MIT License

Copyright (c) 2026 Gulliby-development

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`
	if string(license) != want {
		t.Errorf("LICENSE does not match the complete required MIT text")
	}
}
