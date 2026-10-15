package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Gulliby-development/ecfit/internal/text"
)

// Path pairs a filesystem path with the spelling used in diagnostics.
type Path struct {
	Actual  string
	Display string
}

// ExpandPaths expands path arguments in order. Directory contents are visited
// recursively in bytewise name order; repeated arguments remain repeated.
func ExpandPaths(paths []string) ([]Path, error) {
	files := make([]Path, 0)
	for _, actual := range paths {
		display := text.Display(actual)
		info, err := os.Lstat(actual)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("%s: not found", display)
			}
			return nil, fmt.Errorf("%s: cannot read the directory", display)
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				continue
			}
			if err := expandDirectory(actual, display, &files); err != nil {
				return nil, err
			}
			continue
		}
		if info.Mode().IsRegular() {
			files = append(files, Path{Actual: actual, Display: display})
		}
	}
	return files, nil
}

func expandDirectory(actual, display string, files *[]Path) error {
	entries, err := os.ReadDir(actual)
	if err != nil {
		return fmt.Errorf("%s: cannot read the directory", display)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	parentDisplay := strings.TrimSuffix(display, "/")
	for _, entry := range entries {
		childActual := filepath.Join(actual, entry.Name())
		childDisplay := parentDisplay + "/" + entry.Name()
		info, err := os.Lstat(childActual)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: not found", childDisplay)
			}
			return fmt.Errorf("%s: cannot read the directory", childDisplay)
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				continue
			}
			if err := expandDirectory(childActual, childDisplay, files); err != nil {
				return err
			}
			continue
		}
		if info.Mode().IsRegular() {
			*files = append(*files, Path{Actual: childActual, Display: childDisplay})
		}
	}
	return nil
}
