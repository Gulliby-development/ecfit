package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWalkTestFile(t *testing.T, path, contents string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func walkDisplay(path string) string {
	return filepath.ToSlash(path)
}

func walkChildDisplay(parent, name string) string {
	return strings.TrimSuffix(walkDisplay(parent), "/") + "/" + name
}

func assertWalkPaths(t *testing.T, got, want []Path) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d paths, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		gotReal, err := filepath.EvalSymlinks(got[i].Actual)
		if err != nil {
			t.Fatalf("resolve actual path %q: %v", got[i].Actual, err)
		}
		wantReal, err := filepath.EvalSymlinks(want[i].Actual)
		if err != nil {
			t.Fatalf("resolve expected path %q: %v", want[i].Actual, err)
		}
		if filepath.Clean(gotReal) != filepath.Clean(wantReal) {
			t.Errorf("path %d actual = %q, want resolved %q", i, gotReal, wantReal)
		}
		if got[i].Display != want[i].Display {
			t.Errorf("path %d display = %q, want %q", i, got[i].Display, want[i].Display)
		}
	}
}

func TestExpandPathsBytewiseRecursiveOrder(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, "z.txt"), "z")
	writeWalkTestFile(t, filepath.Join(root, "A.txt"), "A")
	writeWalkTestFile(t, filepath.Join(root, "nested", "b.txt"), "b")
	writeWalkTestFile(t, filepath.Join(root, "nested", "a.txt"), "a")

	got, err := ExpandPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{
		{Actual: filepath.Join(root, "A.txt"), Display: walkChildDisplay(root, "A.txt")},
		{Actual: filepath.Join(root, "nested", "a.txt"), Display: walkChildDisplay(filepath.Join(root, "nested"), "a.txt")},
		{Actual: filepath.Join(root, "nested", "b.txt"), Display: walkChildDisplay(filepath.Join(root, "nested"), "b.txt")},
		{Actual: filepath.Join(root, "z.txt"), Display: walkChildDisplay(root, "z.txt")},
	})
}

func TestExpandPathsDirectoryWithAndWithoutTrailingSeparator(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "d")
	writeWalkTestFile(t, filepath.Join(dir, "a.txt"), "a")
	writeWalkTestFile(t, filepath.Join(dir, "b.txt"), "b")
	want := []Path{
		{Actual: filepath.Join(dir, "a.txt"), Display: walkChildDisplay(dir, "a.txt")},
		{Actual: filepath.Join(dir, "b.txt"), Display: walkChildDisplay(dir, "b.txt")},
	}

	withoutSlash, err := ExpandPaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, withoutSlash, want)

	withSlash, err := ExpandPaths([]string{dir + string(os.PathSeparator)})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, withSlash, want)
}

func TestExpandPathsPreservesTypedFileSpelling(t *testing.T) {
	root := t.TempDir()
	file := writeWalkTestFile(t, filepath.Join(root, "a.txt"), "a")
	typed := filepath.Dir(file) + string(os.PathSeparator) + "." + string(os.PathSeparator) + filepath.Base(file)

	got, err := ExpandPaths([]string{typed})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{{Actual: typed, Display: walkDisplay(typed)}})
}

func TestExpandPathsKeepsRepeatedArgumentsInOrder(t *testing.T) {
	root := t.TempDir()
	first := writeWalkTestFile(t, filepath.Join(root, "first.txt"), "1")
	second := writeWalkTestFile(t, filepath.Join(root, "second.txt"), "2")

	got, err := ExpandPaths([]string{second, first, second})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{
		{Actual: second, Display: walkDisplay(second)},
		{Actual: first, Display: walkDisplay(first)},
		{Actual: second, Display: walkDisplay(second)},
	})
}

func TestExpandPathsSkipsRootGitDirectory(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".git", "config"), "git")
	visible := writeWalkTestFile(t, filepath.Join(root, "visible.txt"), "visible")

	got, err := ExpandPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{{Actual: visible, Display: walkChildDisplay(root, "visible.txt")}})
}

func TestExpandPathsSkipsNestedGitDirectory(t *testing.T) {
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, "sub", ".git", "objects", "item"), "git")
	visible := writeWalkTestFile(t, filepath.Join(root, "sub", "visible.txt"), "visible")

	got, err := ExpandPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{{Actual: visible, Display: walkChildDisplay(filepath.Join(root, "sub"), "visible.txt")}})
}

func TestExpandPathsSkipsGitDirectoryArgument(t *testing.T) {
	git := filepath.Join(t.TempDir(), ".git")
	writeWalkTestFile(t, filepath.Join(git, "config"), "git")

	got, err := ExpandPaths([]string{git})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{})
}

func TestExpandPathsIncludesRegularEditorConfigFiles(t *testing.T) {
	root := t.TempDir()
	config := writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n")
	source := writeWalkTestFile(t, filepath.Join(root, "a.txt"), "a")

	got, err := ExpandPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{
		{Actual: config, Display: walkChildDisplay(root, ".editorconfig")},
		{Actual: source, Display: walkChildDisplay(root, "a.txt")},
	})
}

func TestExpandPathsIncludesNestedRegularConfig(t *testing.T) {
	root := t.TempDir()
	config := writeWalkTestFile(t, filepath.Join(root, "sub", ".editorconfig"), "[*.txt]\n")
	source := writeWalkTestFile(t, filepath.Join(root, "sub", "a.txt"), "a")

	got, err := ExpandPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{
		{Actual: config, Display: walkChildDisplay(filepath.Join(root, "sub"), ".editorconfig")},
		{Actual: source, Display: walkChildDisplay(filepath.Join(root, "sub"), "a.txt")},
	})
}

func TestExpandPathsDoesNotSkipGitNamedFilesOrOtherDirectories(t *testing.T) {
	root := t.TempDir()
	file := writeWalkTestFile(t, filepath.Join(root, ".git"), "regular file")
	other := writeWalkTestFile(t, filepath.Join(root, "project.git", "kept.txt"), "kept")

	got, err := ExpandPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{
		{Actual: file, Display: walkChildDisplay(root, ".git")},
		{Actual: other, Display: walkChildDisplay(filepath.Join(root, "project.git"), "kept.txt")},
	})
}

func TestExpandPathsEmptyDirectoryHasNoFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "empty")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ExpandPaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{})
}

func TestExpandPathsMissingArgument(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")

	got, err := ExpandPaths([]string{missing})
	if err == nil || err.Error() != walkDisplay(missing)+": not found" {
		t.Fatalf("error = %v, want %q", err, walkDisplay(missing)+": not found")
	}
	if got != nil {
		t.Fatalf("paths = %#v, want nil after input error", got)
	}
}

func TestExpandPathsDiscardsEarlierResultsWhenLaterPathIsMissing(t *testing.T) {
	file := writeWalkTestFile(t, filepath.Join(t.TempDir(), "first.txt"), "first")
	missing := filepath.Join(t.TempDir(), "missing.txt")

	got, err := ExpandPaths([]string{file, missing})
	if err == nil || err.Error() != walkDisplay(missing)+": not found" {
		t.Fatalf("error = %v, want %q", err, walkDisplay(missing)+": not found")
	}
	if got != nil {
		t.Fatalf("paths = %#v, want nil after later input error", got)
	}
}

func TestExpandPathsSkipsSymlinkEntriesWhenSupported(t *testing.T) {
	root := t.TempDir()
	target := writeWalkTestFile(t, filepath.Join(root, "target.txt"), "target")
	targetDir := filepath.Join(root, "target-dir")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "inside.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}

	fileLink := filepath.Join(root, "file-link.txt")
	fileLinkCreated := os.Symlink(target, fileLink) == nil
	dirLink := filepath.Join(root, "dir-link")
	dirLinkCreated := os.Symlink(targetDir, dirLink) == nil

	args := []string{root}
	if fileLinkCreated {
		args = append(args, fileLink)
	}
	if dirLinkCreated {
		args = append(args, dirLink)
	}
	got, err := ExpandPaths(args)
	if err != nil {
		t.Fatal(err)
	}
	assertWalkPaths(t, got, []Path{
		{Actual: filepath.Join(targetDir, "inside.txt"), Display: walkChildDisplay(targetDir, "inside.txt")},
		{Actual: target, Display: walkChildDisplay(root, "target.txt")},
	})
}

func TestRunCheckWalksInSortedOrderWithExactOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".editorconfig"), []byte("root = true\n\n[*.txt]\ninsert_final_newline = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "d")
	writeWalkTestFile(t, filepath.Join(dir, "b.txt"), "b")
	writeWalkTestFile(t, filepath.Join(dir, "a.txt"), "a")
	writeWalkTestFile(t, filepath.Join(dir, "sub", "c.txt"), "c")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"check", "--root", root, dir}, &stdout, &stderr)
	expected := walkChildDisplay(dir, "a.txt") + ":1: EC003 missing final newline\n" +
		walkChildDisplay(dir, "b.txt") + ":1: EC003 missing final newline\n" +
		walkChildDisplay(filepath.Join(dir, "sub"), "c.txt") + ":1: EC003 missing final newline\n" +
		"3 problems in 3 files, 3 files checked\n"
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if stdout.String() != expected {
		t.Errorf("stdout = %q, want %q", stdout.String(), expected)
	}
	if stderr.String() != "" {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}
