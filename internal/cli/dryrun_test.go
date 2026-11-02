package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Gulliby-development/ecfit/internal/rules"
)

func TestParseArgsDryRun(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name    string
		argv    []string
		dryRun  bool
		paths   []string
		wantErr string
	}{
		{name: "fix without the option", argv: []string{"fix", "--root", root, "a"}, paths: []string{"a"}},
		{name: "option before the path", argv: []string{"fix", "--dry-run", "--root", root, "a"}, dryRun: true, paths: []string{"a"}},
		{name: "option after the path", argv: []string{"fix", "--root=" + root, "a", "--dry-run"}, dryRun: true, paths: []string{"a"}},
		{name: "option between paths", argv: []string{"fix", "--root", root, "a", "--dry-run", "b"}, dryRun: true, paths: []string{"a", "b"}},
		{name: "repeated option", argv: []string{"fix", "--dry-run", "--root", root, "--dry-run", "a"}, dryRun: true, paths: []string{"a"}},
		{name: "check refuses it", argv: []string{"check", "--root", root, "--dry-run", "a"}, wantErr: `unknown option "--dry-run"`},
		{name: "explain refuses it", argv: []string{"explain", "--dry-run", "--root", root, "a"}, wantErr: `unknown option "--dry-run"`},
		{name: "equals form is unknown", argv: []string{"fix", "--root", root, "--dry-run=1", "a"}, wantErr: `unknown option "--dry-run=1"`},
		{name: "prefix is unknown", argv: []string{"fix", "--root", root, "--dry", "a"}, wantErr: `unknown option "--dry"`},
		{name: "option alone still needs a path", argv: []string{"fix", "--root", root, "--dry-run"}, wantErr: "missing PATH"},
		{name: "root value is not an option", argv: []string{"fix", "--root", "--dry-run", "a"}, wantErr: "--dry-run: not a directory"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseArgs(tc.argv)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("ParseArgs(%q) error = %v, want %q", tc.argv, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseArgs(%q) error = %v", tc.argv, err)
			}
			if got.DryRun != tc.dryRun || !reflect.DeepEqual(got.Paths, tc.paths) {
				t.Fatalf("ParseArgs(%q) = %+v, want DryRun %v and paths %q", tc.argv, got, tc.dryRun, tc.paths)
			}
		})
	}
}

func TestFormatFixDryRun(t *testing.T) {
	trailing := []rules.Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}}
	cases := []struct {
		name  string
		files []FileReport
		want  string
	}{
		{name: "zero files", want: "would fix 0 files, 0 problems left\n"},
		{
			name:  "one file",
			files: []FileReport{{Path: "a.txt", Changed: true, OriginalFindings: trailing}},
			want:  "a.txt: would fix EC002\nwould fix 1 file, 0 problems left\n",
		},
		{
			name: "a problem left and a file kept",
			files: []FileReport{
				{Path: "a.txt", Changed: true, OriginalFindings: trailing, Findings: []rules.Finding{{Line: 3, Code: "EC006", Message: "line is 90 characters long, the limit is 80"}}},
				{Path: "b.txt"},
			},
			want: "a.txt: would fix EC002\na.txt:3: EC006 line is 90 characters long, the limit is 80\nwould fix 1 file, 1 problem left\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertReportText(t, FormatFix(tc.files, true), tc.want)
		})
	}
}

func TestRunFixDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "[*.txt]\nend_of_line = lf\ntrim_trailing_whitespace = true\ninsert_final_newline = true\nindent_style = tab\n")
	a := writeWalkTestFile(t, filepath.Join(root, "a.txt"), "a  \r\nb")
	b := writeWalkTestFile(t, filepath.Join(root, "b.txt"), "ok\n")
	c := writeWalkTestFile(t, filepath.Join(root, "c.txt"), "  indented\n")
	display := func(path string) string { return filepath.ToSlash(path) }

	var stdout, stderr bytes.Buffer
	code := Run([]string{"fix", "--dry-run", "--root", root, a, b, c}, &stdout, &stderr)
	want := display(a) + ": would fix EC001, EC002, EC003\n" +
		display(c) + ":1: EC004 indented with a space, expected tabs\n" +
		"would fix 1 file, 1 problem left\n"
	if code != 1 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("Run = (%d, %q, %q), want (1, %q, empty)", code, stdout.String(), stderr.String(), want)
	}
	for path, original := range map[string]string{a: "a  \r\nb", b: "ok\n", c: "  indented\n"} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != original {
			t.Errorf("%s = %q (%v), want it unchanged: %q", path, data, err, original)
		}
	}

	stdout.Reset()
	code = Run([]string{"fix", "--root", root, a, b, c}, &stdout, &stderr)
	want = strings.ReplaceAll(want, "would fix", "fixed")
	if code != 1 || stdout.String() != want {
		t.Fatalf("Run without the option = (%d, %q), want (1, %q)", code, stdout.String(), want)
	}
	if data, _ := os.ReadFile(a); string(data) != "a\nb\n" {
		t.Errorf("a.txt = %q after fix, want %q", data, "a\nb\n")
	}
}

func TestRunFixDryRunCleanProject(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "[*.txt]\nend_of_line = lf\ninsert_final_newline = true\n")
	a := writeWalkTestFile(t, filepath.Join(root, "a.txt"), "fine\n")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"fix", "--root", root, "--dry-run", a}, &stdout, &stderr)
	if code != 0 || stdout.String() != "would fix 0 files, 0 problems left\n" || stderr.Len() != 0 {
		t.Fatalf("Run = (%d, %q, %q), want (0, %q, empty)", code, stdout.String(), stderr.String(), "would fix 0 files, 0 problems left\n")
	}
}

func TestRunFixDryRunSkipsBinaryFiles(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "[*]\ninsert_final_newline = true\n")
	a := writeWalkTestFile(t, filepath.Join(root, "a.txt"), "no newline")
	bin := writeWalkTestFile(t, filepath.Join(root, "b.bin"), "x\x00y")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"fix", "--dry-run", "--root", root, a, bin}, &stdout, &stderr)
	want := filepath.ToSlash(a) + ": would fix EC003\nwould fix 1 file, 0 problems left, 1 binary file skipped\n"
	if code != 0 || stdout.String() != want {
		t.Fatalf("Run = (%d, %q), want (0, %q)", code, stdout.String(), want)
	}
}
