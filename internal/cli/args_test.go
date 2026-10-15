package cli

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Gulliby-development/ecfit/internal/text"
)

func TestParseArgsPrecedenceAndRoots(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name    string
		argv    func(string) []string
		command string
		root    string
		paths   []string
		help    bool
		wantErr string
	}{
		{name: "missing command", argv: func(string) []string { return nil }, wantErr: "missing command (explain, check or fix)"},
		{name: "unknown command", argv: func(string) []string { return []string{"build"} }, wantErr: `unknown command "build"`},
		{name: "unknown command ARG", argv: func(string) []string { return []string{"ARG"} }, wantErr: `unknown command "ARG"`},
		{name: "unknown command precedes root parsing", argv: func(string) []string { return []string{"build", "--root"} }, wantErr: `unknown command "build"`},
		{name: "only help argument", argv: func(string) []string { return []string{"--help"} }, help: true},
		{name: "help with another argument is an unknown command", argv: func(string) []string { return []string{"--help", "file"} }, wantErr: `unknown command "--help"`},
		{name: "help after command is an unknown option", argv: func(string) []string { return []string{"explain", "--help"} }, wantErr: `unknown option "--help"`},
		{name: "explain requires a file", argv: func(r string) []string { return []string{"explain", "--root", r} }, wantErr: "missing FILE"},
		{name: "check requires a path", argv: func(r string) []string { return []string{"check", "--root", r} }, wantErr: "missing PATH"},
		{name: "fix requires a path", argv: func(r string) []string { return []string{"fix", "--root=" + r} }, wantErr: "missing PATH"},
		{name: "explain accepts one file", argv: func(r string) []string { return []string{"explain", "--root", r, "a", "b"} }, wantErr: "too many arguments"},
		{name: "unknown option is quoted", argv: func(r string) []string { return []string{"check", "--root", r, "-v"} }, wantErr: `unknown option "-v"`},
		{name: "unknown option ARG", argv: func(r string) []string { return []string{"check", "--root", r, "-ARG"} }, wantErr: `unknown option "-ARG"`},
		{name: "root option without value", argv: func(string) []string { return []string{"explain", "--root"} }, wantErr: "--root needs a value"},
		{name: "root option without value precedes missing file", argv: func(r string) []string { return []string{"explain", "--root", r, "--root"} }, wantErr: "--root needs a value"},
		{name: "unknown option precedes missing path", argv: func(r string) []string { return []string{"check", "--root", r, "--bad"} }, wantErr: `unknown option "--bad"`},
		{name: "separated root consumes a dash-prefixed value", argv: func(string) []string { return []string{"explain", "--root", "-dash", "file"} }, wantErr: "-dash: not a directory"},
		{name: "separated root consumes another root-looking value", argv: func(string) []string { return []string{"check", "--root", "--root", "file"} }, wantErr: "--root: not a directory"},
		{name: "equals root accepts an absolute value", argv: func(r string) []string { return []string{"explain", "--root=" + r, "file"} }, command: "explain", root: root, paths: []string{"file"}},
		{name: "separated root accepts an absolute value", argv: func(r string) []string { return []string{"check", "--root", r, "a", "b"} }, command: "check", root: root, paths: []string{"a", "b"}},
		{name: "last separated root wins", argv: func(r string) []string {
			return []string{"fix", "--root", filepath.Join(r, "absent"), "--root", r, "file"}
		}, command: "fix", root: root, paths: []string{"file"}},
		{name: "last equals root wins", argv: func(r string) []string {
			return []string{"explain", "--root", r, "--root=" + filepath.Join(r, "absent"), "file"}
		}, wantErr: text.Display(filepath.Join(root, "absent")) + ": not a directory"},
		{name: "last root may use equals after separated form", argv: func(r string) []string {
			return []string{"check", "--root", filepath.Join(r, "absent"), "--root=" + r, "file"}
		}, command: "check", root: root, paths: []string{"file"}},
		{name: "last root may use separated form after equals", argv: func(r string) []string {
			return []string{"fix", "--root=" + filepath.Join(r, "absent"), "--root", r, "file"}
		}, command: "fix", root: root, paths: []string{"file"}},
		{name: "path may precede root option", argv: func(r string) []string { return []string{"explain", "file", "--root", r} }, command: "explain", root: root, paths: []string{"file"}},
		{name: "check retains multiple paths in order", argv: func(r string) []string { return []string{"check", "a", "--root=" + r, "b"} }, command: "check", root: root, paths: []string{"a", "b"}},
		{name: "fix retains multiple paths in order", argv: func(r string) []string { return []string{"fix", "--root", r, "b", "a"} }, command: "fix", root: root, paths: []string{"b", "a"}},
		{name: "dash-prefixed path is an unknown option", argv: func(r string) []string { return []string{"explain", "--root", r, "-file"} }, wantErr: `unknown option "-file"`},
		{name: "missing path precedes invalid root", argv: func(r string) []string { return []string{"check", "--root", filepath.Join(r, "absent")} }, wantErr: "missing PATH"},
		{name: "too many paths precede invalid root", argv: func(r string) []string { return []string{"explain", "--root", filepath.Join(r, "absent"), "a", "b"} }, wantErr: "too many arguments"},
		{name: "invalid root follows path validation", argv: func(r string) []string { return []string{"explain", "--root", filepath.Join(r, "absent"), "file"} }, wantErr: text.Display(filepath.Join(root, "absent")) + ": not a directory"},
		{name: "directory root is required", argv: func(r string) []string { return []string{"explain", "--root", filepath.Join(r, "file"), "file"} }, wantErr: text.Display(filepath.Join(root, "file")) + ": not a directory"},
		{name: "empty root value is validated as a directory", argv: func(string) []string { return []string{"explain", "--root=", "file"} }, wantErr: ": not a directory"},
		{name: "root with a dash-prefixed equals value is validated", argv: func(string) []string { return []string{"explain", "--root=-dash", "file"} }, wantErr: "-dash: not a directory"},
		{name: "root option can consume a later option as its value", argv: func(string) []string { return []string{"explain", "--root", "--help", "file"} }, wantErr: "--help: not a directory"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseArgs(tc.argv(root))
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("ParseArgs() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseArgs() error = %v", err)
			}
			if got.Command != tc.command || got.Root != tc.root || got.Help != tc.help || !reflect.DeepEqual(got.Paths, tc.paths) {
				t.Fatalf("ParseArgs() = %#v, want command=%q root=%q paths=%#v help=%t", got, tc.command, tc.root, tc.paths, tc.help)
			}
		})
	}
}

func TestRunArgumentOutput(t *testing.T) {
	cases := []struct {
		name       string
		argv       func(string) []string
		wantCode   int
		wantStdout string
		wantStderr func(string) string
	}{
		{
			name:       "help prints exact usage",
			argv:       func(string) []string { return []string{"--help"} },
			wantStdout: "usage: ecfit explain [--root DIR] FILE\n       ecfit check [--root DIR] PATH...\n       ecfit fix [--root DIR] PATH...\n",
			wantStderr: func(string) string { return "" },
		},
		{
			name:       "help with extra argument is not help",
			argv:       func(string) []string { return []string{"--help", "file"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: unknown command \"--help\"\n" },
		},
		{
			name:       "missing command is prefixed",
			argv:       func(string) []string { return nil },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: missing command (explain, check or fix)\n" },
		},
		{
			name:       "unknown command is prefixed",
			argv:       func(string) []string { return []string{"run"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: unknown command \"run\"\n" },
		},
		{
			name:       "unknown command ARG is exact",
			argv:       func(string) []string { return []string{"ARG"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: unknown command \"ARG\"\n" },
		},
		{
			name:       "root value error precedes missing file",
			argv:       func(r string) []string { return []string{"explain", "--root", r, "--root"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: --root needs a value\n" },
		},
		{
			name:       "unknown option has no stdout",
			argv:       func(r string) []string { return []string{"check", "--root", r, "--json"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: unknown option \"--json\"\n" },
		},
		{
			name:       "unknown option ARG is exact",
			argv:       func(r string) []string { return []string{"check", "--root", r, "-ARG"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: unknown option \"-ARG\"\n" },
		},
		{
			name:       "missing explain operand",
			argv:       func(r string) []string { return []string{"explain", "--root=" + r} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: missing FILE\n" },
		},
		{
			name:       "missing check operand",
			argv:       func(r string) []string { return []string{"check", "--root", r} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: missing PATH\n" },
		},
		{
			name:       "too many explain operands",
			argv:       func(r string) []string { return []string{"explain", "--root", r, "a", "b"} },
			wantCode:   2,
			wantStderr: func(string) string { return "ecfit: too many arguments\n" },
		},
		{
			name:     "invalid root message preserves its path",
			argv:     func(r string) []string { return []string{"fix", "--root", filepath.Join(r, "missing"), "file"} },
			wantCode: 2,
			wantStderr: func(r string) string {
				return "ecfit: " + text.Display(filepath.Join(r, "missing")) + ": not a directory\n"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			var stdout, stderr bytes.Buffer
			code := Run(tc.argv(root), &stdout, &stderr)
			if code != tc.wantCode {
				t.Errorf("Run() code = %d, want %d", code, tc.wantCode)
			}
			if got := stdout.String(); got != tc.wantStdout {
				t.Errorf("Run() stdout = %q, want %q", got, tc.wantStdout)
			}
			if got, want := stderr.String(), tc.wantStderr(root); got != want {
				t.Errorf("Run() stderr = %q, want %q", got, want)
			}
		})
	}
}
