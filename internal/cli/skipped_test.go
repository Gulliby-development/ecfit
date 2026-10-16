package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func skippedTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n[*.txt]\ninsert_final_newline = true\n")
	return root
}

func TestRunCheckSkippedBinaryCounts(t *testing.T) {
	t.Run("singular skipped file", func(t *testing.T) {
		root := skippedTestRoot(t)
		binary := writeWalkTestFile(t, filepath.Join(root, "binary.bin"), string([]byte{0}))

		assertRunOutput(t, []string{"check", "--root", root, binary}, 0, "ok, 0 files checked, 1 binary file skipped\n", "")
	})

	t.Run("plural skipped files", func(t *testing.T) {
		root := skippedTestRoot(t)
		first := writeWalkTestFile(t, filepath.Join(root, "first.bin"), string([]byte{0}))
		second := writeWalkTestFile(t, filepath.Join(root, "second.bin"), string([]byte{'x', 0, 'y'}))

		assertRunOutput(t, []string{"check", "--root", root, first, second}, 0, "ok, 0 files checked, 2 binary files skipped\n", "")
	})

	t.Run("binary beside clean file", func(t *testing.T) {
		root := skippedTestRoot(t)
		clean := writeWalkTestFile(t, filepath.Join(root, "clean.txt"), "clean\n")
		binary := writeWalkTestFile(t, filepath.Join(root, "binary.bin"), string([]byte{0}))

		assertRunOutput(t, []string{"check", "--root", root, clean, binary}, 0, "ok, 1 file checked, 1 binary file skipped\n", "")
	})

	t.Run("binary beside file with a problem", func(t *testing.T) {
		root := skippedTestRoot(t)
		problem := writeWalkTestFile(t, filepath.Join(root, "problem.txt"), "missing final newline")
		binary := writeWalkTestFile(t, filepath.Join(root, "binary.bin"), string([]byte{0}))

		want := walkDisplay(problem) + ":1: EC003 missing final newline\n" +
			"1 problem in 1 file, 1 file checked, 1 binary file skipped\n"
		assertRunOutput(t, []string{"check", "--root", root, problem, binary}, 1, want, "")
	})

	t.Run("only skipped files", func(t *testing.T) {
		root := t.TempDir()
		writeWalkTestFile(t, filepath.Join(root, "first.bin"), string([]byte{0}))
		writeWalkTestFile(t, filepath.Join(root, "second.bin"), string([]byte{0}))

		assertRunOutput(t, []string{"check", "--root", root, root}, 0, "ok, 0 files checked, 2 binary files skipped\n", "")
	})

	t.Run("same skipped path given twice", func(t *testing.T) {
		root := skippedTestRoot(t)
		binary := writeWalkTestFile(t, filepath.Join(root, "binary.bin"), string([]byte{0}))

		assertRunOutput(t, []string{"check", "--root", root, binary, binary}, 0, "ok, 0 files checked, 2 binary files skipped\n", "")
	})

	t.Run("directory and file operands count the same skipped path twice", func(t *testing.T) {
		root := skippedTestRoot(t)
		binary := writeWalkTestFile(t, filepath.Join(root, "binary.bin"), string([]byte{0}))

		assertRunOutput(t, []string{"check", "--root", root, root, binary}, 0, "ok, 1 file checked, 2 binary files skipped\n", "")
	})

	t.Run("zero skips preserve check and fix output", func(t *testing.T) {
		root := skippedTestRoot(t)
		clean := writeWalkTestFile(t, filepath.Join(root, "clean.txt"), "clean\n")

		assertRunOutput(t, []string{"check", "--root", root, clean}, 0, "ok, 1 file checked\n", "")
		assertRunOutput(t, []string{"fix", "--root", root, clean}, 0, "fixed 0 files, 0 problems left\n", "")
	})
}

func TestRunFixSkippedBinaryCount(t *testing.T) {
	t.Run("count skipped file and preserve its bytes without writing it", func(t *testing.T) {
		root := skippedTestRoot(t)
		binaryBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x0a}
		binary := writeWalkTestFile(t, filepath.Join(root, "logo.bin"), string(binaryBytes))
		fixable := writeWalkTestFile(t, filepath.Join(root, "fixable.txt"), "needs newline")

		originalWrite := commitWriteFile
		defer func() { commitWriteFile = originalWrite }()
		writes := make([]string, 0)
		commitWriteFile = func(path string, data []byte, mode os.FileMode) error {
			writes = append(writes, path)
			return os.WriteFile(path, data, mode)
		}

		want := walkDisplay(fixable) + ": fixed EC003\nfixed 1 file, 0 problems left, 1 binary file skipped\n"
		assertRunOutput(t, []string{"fix", "--root", root, binary, fixable}, 0, want, "")
		assertFileContent(t, binary, binaryBytes)
		assertFileContent(t, fixable, []byte("needs newline\n"))
		if len(writes) != 1 || filepath.Clean(writes[0]) != filepath.Clean(fixable) {
			t.Errorf("commit writes = %#v, want only %q", writes, fixable)
		}
	})
}

func TestRunWorkedDirectoryAndExplainWithSkippedFiles(t *testing.T) {
	t.Run("worked check fix check sequence includes editorconfig", func(t *testing.T) {
		root := t.TempDir()
		writeWalkTestFile(t, filepath.Join(root, ".editorconfig"), "root = true\n\n[*]\ninsert_final_newline = true\n")
		writeWalkTestFile(t, filepath.Join(root, "a.txt"), "a\n")
		b := writeWalkTestFile(t, filepath.Join(root, "b.txt"), "b")
		logoBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x0a}
		logo := writeWalkTestFile(t, filepath.Join(root, "logo.bin"), string(logoBytes))
		dataBytes := []byte{0}
		data := writeWalkTestFile(t, filepath.Join(root, "data.bin"), string(dataBytes))
		bPath := walkChildDisplay(root, "b.txt")

		checkWant := bPath + ":1: EC003 missing final newline\n" +
			"1 problem in 1 file, 3 files checked, 2 binary files skipped\n"
		assertRunOutput(t, []string{"check", "--root", root, root}, 1, checkWant, "")

		fixWant := bPath + ": fixed EC003\n" +
			"fixed 1 file, 0 problems left, 2 binary files skipped\n"
		assertRunOutput(t, []string{"fix", "--root", root, root}, 0, fixWant, "")
		assertFileContent(t, b, []byte("b\n"))
		assertFileContent(t, logo, logoBytes)
		assertFileContent(t, data, dataBytes)

		assertRunOutput(t, []string{"check", "--root", root, root}, 0, "ok, 3 files checked, 2 binary files skipped\n", "")
	})

	t.Run("explain is unchanged and skipped report uses LF", func(t *testing.T) {
		root := skippedTestRoot(t)
		binary := writeWalkTestFile(t, filepath.Join(root, "binary.bin"), string([]byte{0}))

		assertRunOutput(t, []string{"explain", "--root", root, binary}, 0, walkDisplay(binary)+"\n  (no settings)\n", "")

		var stdout, stderr strings.Builder
		code := Run([]string{"check", "--root", root, binary}, &stdout, &stderr)
		if code != 0 || stderr.String() != "" {
			t.Fatalf("Run check = (%d, %q, %q), want (0, report, empty)", code, stdout.String(), stderr.String())
		}
		assertReportText(t, stdout.String(), "ok, 0 files checked, 1 binary file skipped\n")
	})
}
