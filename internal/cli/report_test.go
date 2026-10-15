package cli

import (
	"strings"
	"testing"

	"github.com/Gulliby-development/ecfit/internal/config"
	"github.com/Gulliby-development/ecfit/internal/rules"
)

func assertReportText(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("formatted text = %q, want %q", got, want)
	}
	if strings.Contains(got, "\r") {
		t.Errorf("formatted text contains CR; output must use LF only: %q", got)
	}
}

func TestFormatHelpExactLF(t *testing.T) {
	assertReportText(t, FormatHelp(), "usage: ecfit explain [--root DIR] FILE\n       ecfit check [--root DIR] PATH...\n       ecfit fix [--root DIR] PATH...\n")
}

func TestFormatExplainExactSettings(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		settings map[string]config.Setting
		want     string
	}{
		{
			name: "no settings and backslash display",
			path: `src\plain.txt`,
			want: "src/plain.txt\n  (no settings)\n",
		},
		{
			name: "single setting",
			path: "plain.txt",
			settings: map[string]config.Setting{
				"end_of_line": {Value: "lf", File: ".editorconfig", Line: 3},
			},
			want: "plain.txt\n  end_of_line = lf  .editorconfig:3\n",
		},
		{
			name: "key order alignment and provenance",
			path: "src/file.go",
			settings: map[string]config.Setting{
				"max_line_length": {Value: "80", File: "sub/config.ini", Line: 9},
				"indent_style":    {Value: "space", File: ".editorconfig", Line: 1},
				"indent_size":     {Value: "4", File: ".editorconfig", Line: 2},
			},
			want: "src/file.go\n" +
				"  indent_size = 4" + strings.Repeat(" ", 7) + ".editorconfig:2\n" +
				"  indent_style = space  .editorconfig:1\n" +
				"  max_line_length = 80  sub/config.ini:9\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertReportText(t, FormatExplain(tc.path, tc.settings), tc.want)
		})
	}
}

func TestFormatCheckExactTotalsAndMessages(t *testing.T) {
	cases := []struct {
		name    string
		files   []FileReport
		checked int
		want    string
	}{
		{
			name:    "zero files",
			checked: 0,
			want:    "ok, 0 files checked\n",
		},
		{
			name:    "one checked file",
			files:   []FileReport{{Path: "clean.txt"}},
			checked: 1,
			want:    "ok, 1 file checked\n",
		},
		{
			name: "many checked files",
			files: []FileReport{
				{Path: "one.txt"},
				{Path: "two.txt"},
				{Path: "three.txt"},
			},
			checked: 3,
			want:    "ok, 3 files checked\n",
		},
		{
			name: "one diagnostic and singular totals",
			files: []FileReport{{
				Path:     `src\a.txt`,
				Findings: []rules.Finding{{Line: 2, Code: "EC002", Message: "trailing whitespace"}},
			}},
			checked: 1,
			want:    "src/a.txt:2: EC002 trailing whitespace\n1 problem in 1 file, 1 file checked\n",
		},
		{
			name: "multiple diagnostics preserve finding order",
			files: []FileReport{{
				Path: "a.txt",
				Findings: []rules.Finding{
					{Line: 4, Code: "EC002", Message: "trailing whitespace"},
					{Line: 1, Code: "EC001", Message: "line ends with crlf, expected lf"},
				},
			}},
			checked: 1,
			want:    "a.txt:4: EC002 trailing whitespace\na.txt:1: EC001 line ends with crlf, expected lf\n2 problems in 1 file, 1 file checked\n",
		},
		{
			name: "problems in multiple files with plural totals",
			files: []FileReport{
				{Path: "a.txt", Findings: []rules.Finding{{Line: 1, Code: "EC003", Message: "missing final newline"}}},
				{Path: "b.txt", Findings: []rules.Finding{{Line: 3, Code: "EC002", Message: "trailing whitespace"}}},
			},
			checked: 5,
			want:    "a.txt:1: EC003 missing final newline\nb.txt:3: EC002 trailing whitespace\n2 problems in 2 files, 5 files checked\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertReportText(t, FormatCheck(tc.files, tc.checked), tc.want)
		})
	}
}

func TestFormatFixExactTotalsAndCodes(t *testing.T) {
	cases := []struct {
		name  string
		files []FileReport
		want  string
	}{
		{
			name: "zero files",
			want: "fixed 0 files, 0 problems left\n",
		},
		{
			name: "one file fixed with no findings left",
			files: []FileReport{{
				Path:             "a.txt",
				Changed:          true,
				OriginalFindings: []rules.Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}},
			}},
			want: "a.txt: fixed EC002\nfixed 1 file, 0 problems left\n",
		},
		{
			name: "many files fixed and one problem left",
			files: []FileReport{
				{
					Path:             "a.txt",
					Changed:          true,
					OriginalFindings: []rules.Finding{{Line: 1, Code: "EC001", Message: "line ends with crlf, expected lf"}},
				},
				{
					Path:             "b.txt",
					Changed:          true,
					OriginalFindings: []rules.Finding{{Line: 1, Code: "EC007", Message: "missing byte order mark"}},
					Findings:         []rules.Finding{{Line: 2, Code: "EC004", Message: "indented with a space, expected tabs"}},
				},
			},
			want: "a.txt: fixed EC001\nb.txt: fixed EC007\nb.txt:2: EC004 indented with a space, expected tabs\nfixed 2 files, 1 problem left\n",
		},
		{
			name: "fix codes are sorted and deduplicated",
			files: []FileReport{{
				Path:    "many.txt",
				Changed: true,
				OriginalFindings: []rules.Finding{
					{Line: 1, Code: "EC003", Message: "missing final newline"},
					{Line: 2, Code: "EC002", Message: "trailing whitespace"},
					{Line: 3, Code: "EC001", Message: "line ends with crlf, expected lf"},
					{Line: 4, Code: "EC002", Message: "trailing whitespace"},
					{Line: 1, Code: "EC007", Message: "unexpected byte order mark"},
				},
			}},
			want: "many.txt: fixed EC001, EC002, EC003, EC007\nfixed 1 file, 0 problems left\n",
		},
		{
			name: "invalid UTF-8 is not listed as a fix code",
			files: []FileReport{{
				Path:    "invalid.txt",
				Changed: true,
				OriginalFindings: []rules.Finding{
					{Line: 1, Code: "EC001", Message: "line ends with crlf, expected lf"},
					{Line: 2, Code: "EC007", Message: "invalid UTF-8"},
				},
			}},
			want: "invalid.txt: fixed EC001\nfixed 1 file, 0 problems left\n",
		},
		{
			name: "unchanged file has no fixed notice",
			files: []FileReport{{
				Path:             "same.txt",
				OriginalFindings: []rules.Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}},
				Findings:         []rules.Finding{{Line: 1, Code: "EC002", Message: "trailing whitespace"}},
			}},
			want: "same.txt:1: EC002 trailing whitespace\nfixed 0 files, 1 problem left\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertReportText(t, FormatFix(tc.files), tc.want)
		})
	}
}
