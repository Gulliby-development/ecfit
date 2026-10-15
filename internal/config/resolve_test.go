package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type resolveConfigFixture struct {
	path    string
	content string
}

func TestResolveTemporaryTrees(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		configs      []resolveConfigFixture
		configDirs   []string
		parentConfig string
		sibling      bool
		outsideError bool
		want         map[string]Setting
		wantErr      string
	}{
		{
			name:    "root config is included for a root-level file",
			file:    "a.txt",
			configs: []resolveConfigFixture{{".editorconfig", "root = true\n[*]\nindent_size = 2\n"}},
			want:    map[string]Setting{"indent_size": {Value: "2", File: ".editorconfig", Line: 3}},
		},
		{
			name:    "root config is included for a nested file",
			file:    "sub/a.txt",
			configs: []resolveConfigFixture{{".editorconfig", "[*.txt]\nend_of_line = crlf\n"}},
			want:    map[string]Setting{"end_of_line": {Value: "crlf", File: ".editorconfig", Line: 2}},
		},
		{
			name:    "nonexistent file can still receive settings",
			file:    "missing.txt",
			configs: []resolveConfigFixture{{".editorconfig", "[*]\ncharset = utf-8\n"}},
			want:    map[string]Setting{"charset": {Value: "utf-8", File: ".editorconfig", Line: 2}},
		},
		{
			name: "no configs produce an empty settings map",
			file: "a.txt",
			want: map[string]Setting{},
		},
		{
			name:    "unmatched sections produce no settings",
			file:    "a.txt",
			configs: []resolveConfigFixture{{".editorconfig", "[*.go]\nindent_style = tab\n"}},
			want:    map[string]Setting{},
		},
		{
			name: "root and child settings merge from farthest to nearest",
			file: "sub/a.go",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nmax_line_length = 80\nindent_size = 2\n"},
				{"sub/.editorconfig", "[*.go]\nindent_style = tab\n"},
			},
			want: map[string]Setting{
				"indent_size":     {Value: "2", File: ".editorconfig", Line: 3},
				"indent_style":    {Value: "tab", File: "sub/.editorconfig", Line: 2},
				"max_line_length": {Value: "80", File: ".editorconfig", Line: 2},
			},
		},
		{
			name: "nearest config overrides a root value and provenance",
			file: "sub/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_size = 2\n"},
				{"sub/.editorconfig", "[*]\nindent_size = 4\n"},
			},
			want: map[string]Setting{"indent_size": {Value: "4", File: "sub/.editorconfig", Line: 2}},
		},
		{
			name: "later matching section wins within one config",
			file: "a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_size = 2\n[a.txt]\nindent_size = 4\n"},
			},
			want: map[string]Setting{"indent_size": {Value: "4", File: ".editorconfig", Line: 4}},
		},
		{
			name: "later nonmatching section does not override",
			file: "a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_size = 4\n[*.go]\nindent_size = 8\n"},
			},
			want: map[string]Setting{"indent_size": {Value: "4", File: ".editorconfig", Line: 2}},
		},
		{
			name: "nested config glob is relative to its directory",
			file: "sub/pkg/a.go",
			configs: []resolveConfigFixture{
				{"sub/.editorconfig", "[pkg/*.go]\nindent_style = space\n"},
			},
			want: map[string]Setting{"indent_style": {Value: "space", File: "sub/.editorconfig", Line: 2}},
		},
		{
			name: "unset removes inherited key but preserves other settings",
			file: "sub/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_style = space\nindent_size = 2\n"},
				{"sub/.editorconfig", "[a.txt]\nindent_style = unset\n"},
			},
			want: map[string]Setting{"indent_size": {Value: "2", File: ".editorconfig", Line: 3}},
		},
		{
			name: "unmatched unset does not remove inherited key",
			file: "sub/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_style = tab\n"},
				{"sub/.editorconfig", "[*.go]\nindent_style = unset\n"},
			},
			want: map[string]Setting{"indent_style": {Value: "tab", File: ".editorconfig", Line: 2}},
		},
		{
			name: "last repeated assignment retains its source line",
			file: "a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_size = 2\nindent_size = 6\n"},
			},
			want: map[string]Setting{"indent_size": {Value: "6", File: ".editorconfig", Line: 3}},
		},
		{
			name: "nearest root true prevents reading broken root config",
			file: "sub/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_style = tabs\n"},
				{"sub/.editorconfig", "root = true\n[*]\nindent_style = tab\n"},
			},
			want: map[string]Setting{"indent_style": {Value: "tab", File: "sub/.editorconfig", Line: 3}},
		},
		{
			name: "nested root true stops above its directory",
			file: "sub/deep/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_style = tabs\n"},
				{"sub/.editorconfig", "root = true\n[*]\nindent_size = 5\n"},
			},
			want: map[string]Setting{"indent_size": {Value: "5", File: "sub/.editorconfig", Line: 3}},
		},
		{
			name:    "reached broken config errors when no section matches",
			file:    "a.txt",
			configs: []resolveConfigFixture{{".editorconfig", "[*.go]\nindent_style = tabs\n"}},
			wantErr: ".editorconfig:2: invalid value \"tabs\" for indent_style",
		},
		{
			name: "broken ancestor is still read after a valid child config",
			file: "sub/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*]\nindent_size = 0\n"},
				{"sub/.editorconfig", "[*]\nindent_size = 4\n"},
			},
			wantErr: ".editorconfig:2: invalid value \"0\" for indent_size",
		},
		{
			name:       "directory named editorconfig cannot be read as a file",
			file:       "a.txt",
			configDirs: []string{".editorconfig"},
			wantErr:    ".editorconfig: cannot read the file",
		},
		{
			name:         "file outside root is rejected",
			file:         "../outside.txt",
			configs:      []resolveConfigFixture{{".editorconfig", "[*]\\nindent_size = 2\\n"}},
			outsideError: true,
		},
		{
			name:         "sibling prefix is outside root",
			file:         "a.txt",
			sibling:      true,
			outsideError: true,
		},
		{
			name:         "config above root is never searched",
			file:         "a.txt",
			parentConfig: "[*]\nindent_size = 0\n",
			want:         map[string]Setting{},
		},
		{
			name: "config from root applies through nested directories",
			file: "one/two/three/a.txt",
			configs: []resolveConfigFixture{
				{".editorconfig", "[*.txt]\ntrim_trailing_whitespace = true\n"},
			},
			want: map[string]Setting{"trim_trailing_whitespace": {Value: "true", File: ".editorconfig", Line: 2}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "root")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, config := range test.configs {
				writeResolveFixture(t, root, config.path, config.content)
			}
			for _, rel := range test.configDirs {
				if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if test.parentConfig != "" {
				if err := os.WriteFile(filepath.Join(base, ".editorconfig"), []byte(test.parentConfig), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			file := filepath.Join(root, test.file)
			if test.sibling {
				file = filepath.Join(base, "root-sibling", "a.txt")
			}
			got, err := Resolve(root, file)
			if test.outsideError {
				wantErr := filepath.ToSlash(file) + ": outside the root"
				if err == nil || err.Error() != wantErr {
					t.Fatalf("Resolve() error = %v, want %q", err, wantErr)
				}
				return
			}
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("Resolve() error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v, want nil", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Resolve() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func writeResolveFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
