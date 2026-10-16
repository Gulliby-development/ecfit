package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunIntegrationCases(t *testing.T) {
	root := t.TempDir()
	existing := writeWalkTestFile(t, filepath.Join(root, "existing.txt"), "plain\n")
	rootFile := writeWalkTestFile(t, filepath.Join(root, "not-dir"), "file")
	missingRoot := filepath.Join(root, "missing-root")
	outside := writeWalkTestFile(t, filepath.Join(t.TempDir(), "outside.txt"), "outside")
	emptyDir := filepath.Join(root, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	brokenDir := filepath.Join(root, "broken")
	writeWalkTestFile(t, filepath.Join(brokenDir, ".editorconfig"), "[*.txt]\nindent_style = tabs\n")
	brokenSyntax := filepath.Join(root, "syntax")
	writeWalkTestFile(t, filepath.Join(brokenSyntax, ".editorconfig"), "[*.txt]\nnot an assignment\n")
	configDir := filepath.Join(root, "config-dir")
	if err := os.MkdirAll(filepath.Join(configDir, ".editorconfig"), 0o755); err != nil {
		t.Fatal(err)
	}
	findingDir := filepath.Join(root, "findings")
	writeWalkTestFile(t, filepath.Join(findingDir, ".editorconfig"), "[*.txt]\nend_of_line = lf\ntrim_trailing_whitespace = true\ninsert_final_newline = true\n")
	finding := writeWalkTestFile(t, filepath.Join(findingDir, "a.txt"), "a  \r\nb")

	display := func(path string) string { return strings.ReplaceAll(path, "\\", "/") }
	errorText := func(message string) string { return "ecfit: " + message + "\n" }
	help := "usage: ecfit explain [--root DIR] FILE\n       ecfit check [--root DIR] PATH...\n       ecfit fix [--root DIR] PATH...\n"
	cases := []struct {
		name, stdout, stderr string
		code                 int
		args                 []string
	}{
		{name: "standalone help", args: []string{"--help"}, code: 0, stdout: help},
		{name: "missing command", code: 2, stderr: errorText("missing command (explain, check or fix)")},
		{name: "unknown command", args: []string{"build"}, code: 2, stderr: errorText("unknown command \"build\"")},
		{name: "help with another argument", args: []string{"--help", "file"}, code: 2, stderr: errorText("unknown command \"--help\"")},
		{name: "help after command", args: []string{"check", "--help"}, code: 2, stderr: errorText("unknown option \"--help\"")},
		{name: "root missing value", args: []string{"explain", "--root"}, code: 2, stderr: errorText("--root needs a value")},
		{name: "unknown option", args: []string{"check", "--root", root, "-x"}, code: 2, stderr: errorText("unknown option \"-x\"")},
		{name: "missing explain file", args: []string{"explain", "--root", root}, code: 2, stderr: errorText("missing FILE")},
		{name: "missing check path", args: []string{"check", "--root", root}, code: 2, stderr: errorText("missing PATH")},
		{name: "missing fix path", args: []string{"fix", "--root", root}, code: 2, stderr: errorText("missing PATH")},
		{name: "too many explain paths", args: []string{"explain", "--root", root, "a", "b"}, code: 2, stderr: errorText("too many arguments")},
		{name: "operand error precedes invalid root", args: []string{"explain", "--root", missingRoot}, code: 2, stderr: errorText("missing FILE")},
		{name: "option error precedes invalid root", args: []string{"check", "--root", missingRoot, "--bad", "a"}, code: 2, stderr: errorText("unknown option \"--bad\"")},
		{name: "last root wins", args: []string{"explain", "--root", missingRoot, "--root=" + root, existing}, code: 0, stdout: display(existing) + "\n  (no settings)\n"},
		{name: "root must be directory", args: []string{"explain", "--root", rootFile, existing}, code: 2, stderr: errorText(display(rootFile) + ": not a directory")},
		{name: "outside root", args: []string{"explain", "--root", root, outside}, code: 2, stderr: errorText(display(outside) + ": outside the root")},
		{name: "missing path", args: []string{"check", "--root", root, filepath.Join(root, "missing.txt")}, code: 2, stderr: errorText(display(filepath.Join(root, "missing.txt")) + ": not found")},
		{name: "reached invalid property", args: []string{"explain", "--root", root, filepath.Join(brokenDir, "a.txt")}, code: 2, stderr: errorText("broken/.editorconfig:2: invalid value \"tabs\" for indent_style")},
		{name: "reached malformed assignment", args: []string{"explain", "--root", root, filepath.Join(brokenSyntax, "a.txt")}, code: 2, stderr: errorText("syntax/.editorconfig:2: expected \"[GLOB]\" or \"KEY = VALUE\"")},
		{name: "config directory is unreadable config", args: []string{"explain", "--root", root, filepath.Join(configDir, "a.txt")}, code: 2, stderr: errorText("config-dir/.editorconfig: cannot read the file")},
		{name: "explain with no settings", args: []string{"explain", "--root", root, existing}, code: 0, stdout: display(existing) + "\n  (no settings)\n"},
		{name: "check empty directory", args: []string{"check", "--root", root, emptyDir}, code: 0, stdout: "ok, 0 files checked\n"},
		{name: "check clean file", args: []string{"check", "--root", root, existing}, code: 0, stdout: "ok, 1 file checked\n"},
		{name: "check findings and ordering", args: []string{"check", "--root", root, finding}, code: 1, stdout: display(finding) + ":1: EC001 line ends with crlf, expected lf\n" + display(finding) + ":1: EC002 trailing whitespace\n" + display(finding) + ":2: EC003 missing final newline\n3 problems in 1 file, 1 file checked\n"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assertRunOutput(t, test.args, test.code, test.stdout, test.stderr)
		})
	}
}

func assertRunOutput(t *testing.T, args []string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != wantCode || stdout.String() != wantOut || stderr.String() != wantErr {
		t.Errorf("Run = (%d, %q, %q), want (%d, %q, %q)", code, stdout.String(), stderr.String(), wantCode, wantOut, wantErr)
	}
}

func assertFileContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("%s = %q, %v; want %q", path, got, err, want)
	}
}

func TestRunFixInputFailureDoesNotWriteEarlierFiles(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n[*.txt]\ninsert_final_newline = true\n")
	first := writeWalkTestFile(t, filepath.Join(root, "first.txt"), "first")
	outside := writeWalkTestFile(t, filepath.Join(t.TempDir(), "outside.txt"), "outside")
	wantErr := "ecfit: " + strings.ReplaceAll(outside, "\\", "/") + ": outside the root\n"
	assertRunOutput(t, []string{"fix", "--root", root, first, outside}, 2, "", wantErr)
	assertFileContent(t, first, []byte("first"))
}

func TestRunFixPreparationFailureDoesNotWriteEarlierFiles(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n[*.txt]\ninsert_final_newline = true\n")
	first := writeWalkTestFile(t, filepath.Join(root, "first.txt"), "first")
	broken := filepath.Join(root, "later")
	writeWalkTestFile(t, filepath.Join(broken, ".editorconfig"), "[*.txt]\nindent_style = tabs\n")
	later := writeWalkTestFile(t, filepath.Join(broken, "later.txt"), "later")
	wantErr := "ecfit: later/.editorconfig:2: invalid value \"tabs\" for indent_style\n"
	assertRunOutput(t, []string{"fix", "--root", root, first, later}, 2, "", wantErr)
	assertFileContent(t, first, []byte("first"))
	assertFileContent(t, later, []byte("later"))
}

func TestRunFixCommitFailureRestoresBytesAndSuppressesOutput(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n[*.txt]\ninsert_final_newline = true\n")
	first := writeWalkTestFile(t, filepath.Join(root, "first.txt"), "first")
	second := writeWalkTestFile(t, filepath.Join(root, "second.txt"), "second")
	originalWrite := commitWriteFile
	defer func() { commitWriteFile = originalWrite }()
	calls := 0
	commitWriteFile = func(path string, data []byte, mode os.FileMode) error {
		calls++
		if calls == 2 {
			return errors.New("controlled write failure")
		}
		return os.WriteFile(path, data, mode)
	}
	wantErr := "ecfit: " + strings.ReplaceAll(second, "\\", "/") + ": cannot write the file\n"
	assertRunOutput(t, []string{"fix", "--root", root, first, second}, 2, "", wantErr)
	if calls != 2 {
		t.Errorf("commit writes = %d, want 2", calls)
	}
	assertFileContent(t, first, []byte("first"))
	assertFileContent(t, second, []byte("second"))
}

func TestRunFixDoesNotWriteUnchangedFile(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n[*.txt]\ncharset = utf-8-bom\ninsert_final_newline = true\n")
	file := writeWalkTestFile(t, filepath.Join(root, "bom-only.txt"), string([]byte{0xef, 0xbb, 0xbf}))
	originalWrite := commitWriteFile
	defer func() { commitWriteFile = originalWrite }()
	calls := 0
	commitWriteFile = func(string, []byte, os.FileMode) error { calls++; return nil }
	assertRunOutput(t, []string{"fix", "--root", root, file}, 0, "fixed 0 files, 0 problems left\n", "")
	assertFileContent(t, file, []byte{0xef, 0xbb, 0xbf})
	if calls != 0 {
		t.Errorf("commit writes = %d, want 0", calls)
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	module, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	name := "ecfit"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	executable := filepath.Join(t.TempDir(), name)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-o", executable, ".")
	command.Dir = module
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	return executable
}

func runBuiltCLI(t *testing.T, executable, directory string, args ...string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = directory
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("ecfit timed out: %v", ctx.Err())
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("start ecfit: %v", err)
		}
		code = exit.ExitCode()
	}
	return code, stdout.String(), stderr.String()
}

func TestBuiltCLIExamplesAndRawBytes(t *testing.T) {
	executable := buildCLI(t)
	project := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "root = true\n\n[*]\nend_of_line = lf\ninsert_final_newline = true\ntrim_trailing_whitespace = true\ncharset = utf-8\n\n[*.{go,mod}]\nindent_style = tab\n\n[*.{js,json,yml}]\nindent_style = space\nindent_size = 2\n\n[Makefile]\nindent_style = tab\n\n[docs/**.md]\nmax_line_length = 80\ntrim_trailing_whitespace = false\n"
	writeWalkTestFile(t, filepath.Join(project, ".editorconfig"), config)
	writeWalkTestFile(t, filepath.Join(project, "main.go"), "package main\n\nfunc main() {\n\tprintln(1)  \n}\n")
	writeWalkTestFile(t, filepath.Join(project, "util.go"), "package main\n\nfunc f() {\n    return\n}")
	writeWalkTestFile(t, filepath.Join(project, "Makefile"), "all:\n\techo hi\n")
	writeWalkTestFile(t, filepath.Join(project, "data.json"), "\uFEFF{\n  \"a\": 1\n}\n")
	writeWalkTestFile(t, filepath.Join(project, "docs", "guide.md"), "# Guide\n\nShort.  \nThis line is long enough to pass eighty characters easily, yes it is quite long indeed.\r\n")
	writeWalkTestFile(t, filepath.Join(project, "web", "app.js"), "function f() {\n  var a = 1;\n   var b = 2;\n\tvar c = 3;\n// xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n}\n")
	writeWalkTestFile(t, filepath.Join(project, "web", ".editorconfig"), "[*.js]\nindent_size = 4\nmax_line_length = 60\n")

	explain := "web/app.js\n  charset = utf-8                  .editorconfig:7\n  end_of_line = lf                 .editorconfig:4\n  indent_size = 4                  web/.editorconfig:2\n  indent_style = space             .editorconfig:13\n  insert_final_newline = true      .editorconfig:5\n  max_line_length = 60             web/.editorconfig:3\n  trim_trailing_whitespace = true  .editorconfig:6\n"
	check := "./data.json:1: EC007 unexpected byte order mark\n./docs/guide.md:4: EC001 line ends with crlf, expected lf\n./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80\n./main.go:4: EC002 trailing whitespace\n./util.go:4: EC004 indented with a space, expected tabs\n./util.go:5: EC003 missing final newline\n./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4\n./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4\n./web/app.js:4: EC004 indented with a tab, expected spaces\n./web/app.js:5: EC006 line is 63 characters long, the limit is 60\n10 problems in 5 files, 8 files checked\n"
	fix := "./data.json: fixed EC007\n./docs/guide.md: fixed EC001\n./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80\n./main.go: fixed EC002\n./util.go: fixed EC003\n./util.go:4: EC004 indented with a space, expected tabs\n./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4\n./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4\n./web/app.js:4: EC004 indented with a tab, expected spaces\n./web/app.js:5: EC006 line is 63 characters long, the limit is 60\nfixed 4 files, 6 problems left\n"
	repeat := "./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80\n./util.go:4: EC004 indented with a space, expected tabs\n./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4\n./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4\n./web/app.js:4: EC004 indented with a tab, expected spaces\n./web/app.js:5: EC006 line is 63 characters long, the limit is 60\nfixed 0 files, 6 problems left\n"
	if code, out, err := runBuiltCLI(t, executable, project, "explain", "--root", project, "web/app.js"); code != 0 || out != explain || err != "" {
		t.Fatalf("explain = (%d, %q, %q)", code, out, err)
	}
	if code, out, err := runBuiltCLI(t, executable, project, "check", "--root", project, "."); code != 1 || out != check || err != "" {
		t.Fatalf("check = (%d, %q, %q)", code, out, err)
	}
	if code, out, err := runBuiltCLI(t, executable, project, "fix", "--root", project, "."); code != 1 || out != fix || err != "" {
		t.Fatalf("fix = (%d, %q, %q)", code, out, err)
	}
	for _, item := range []struct {
		name string
		want []byte
	}{
		{"data.json", []byte("{\n  \"a\": 1\n}\n")},
		{"docs/guide.md", []byte("# Guide\n\nShort.  \nThis line is long enough to pass eighty characters easily, yes it is quite long indeed.\n")},
		{"main.go", []byte("package main\n\nfunc main() {\n\tprintln(1)\n}\n")},
		{"util.go", []byte("package main\n\nfunc f() {\n    return\n}\n")},
		{"Makefile", []byte("all:\n\techo hi\n")},
		{".editorconfig", []byte(config)},
		{"web/.editorconfig", []byte("[*.js]\nindent_size = 4\nmax_line_length = 60\n")},
		{"web/app.js", []byte("function f() {\n  var a = 1;\n   var b = 2;\n\tvar c = 3;\n// xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n}\n")},
	} {
		got, err := os.ReadFile(filepath.Join(project, filepath.FromSlash(item.name)))
		if err != nil || !bytes.Equal(got, item.want) {
			t.Errorf("%s = %q, %v; want %q", item.name, got, err, item.want)
		}
	}
	if code, out, err := runBuiltCLI(t, executable, project, "fix", "--root", project, "."); code != 1 || out != repeat || err != "" {
		t.Fatalf("repeat fix = (%d, %q, %q)", code, out, err)
	}

	invalidRoot := filepath.Join(t.TempDir(), "invalid")
	writeWalkTestFile(t, filepath.Join(invalidRoot, ".editorconfig"), string([]byte{'[', '*', ']', '\n', 'x', '=', 0xff}))
	content := writeWalkTestFile(t, filepath.Join(invalidRoot, "file.txt"), "x\n")
	if code, out, err := runBuiltCLI(t, executable, project, "explain", "--root", invalidRoot, content); code != 2 || out != "" || err != "ecfit: .editorconfig: input is not valid UTF-8\n" {
		t.Fatalf("invalid UTF-8 config = (%d, %q, %q)", code, out, err)
	}

	bomRoot := filepath.Join(t.TempDir(), "bom")
	writeWalkTestFile(t, filepath.Join(bomRoot, ".editorconfig"), "root = true\n[*.txt]\ncharset = utf-8\n")
	bomFile := filepath.Join(bomRoot, "bom.txt")
	if err := os.WriteFile(bomFile, []byte{0xef, 0xbb, 0xbf, 'a'}, 0o600); err != nil {
		t.Fatal(err)
	}
	wantBOMFix := strings.ReplaceAll(bomFile, "\\", "/") + ": fixed EC007\nfixed 1 file, 0 problems left\n"
	if code, out, err := runBuiltCLI(t, executable, project, "fix", "--root", bomRoot, bomFile); code != 0 || out != wantBOMFix || err != "" {
		t.Fatalf("BOM fix = (%d, %q, %q)", code, out, err)
	}
	if got, err := os.ReadFile(bomFile); err != nil || !bytes.Equal(got, []byte{'a'}) {
		t.Fatalf("fixed BOM bytes = %v, %v", got, err)
	}

	nulRoot := filepath.Join(t.TempDir(), "binary-root")
	writeWalkTestFile(t, filepath.Join(nulRoot, ".editorconfig"), "root = true\n[*.txt]\ninsert_final_newline = true\n")
	nulFile := filepath.Join(nulRoot, "binary.txt")
	nulBytes := []byte{'a', 0, 'b'}
	if err := os.WriteFile(nulFile, nulBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, err := runBuiltCLI(t, executable, project, "check", "--root", nulRoot, nulFile); code != 0 || out != "ok, 0 files checked, 1 binary file skipped\n" || err != "" {
		t.Fatalf("NUL check = (%d, %q, %q)", code, out, err)
	}
	if got, err := os.ReadFile(nulFile); err != nil || !bytes.Equal(got, nulBytes) {
		t.Fatalf("NUL bytes = %v, %v", got, err)
	}
}
