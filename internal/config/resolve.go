package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gulliby-development/ecfit/internal/text"
)

// Resolve discovers and merges the EditorConfig settings that apply to file.
// The file need not exist, but its absolute path must be inside root.
func Resolve(root, file string) (map[string]Setting, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%s: outside the root", text.Display(file))
	}
	filePath, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("%s: outside the root", text.Display(file))
	}
	rootPath = filepath.Clean(rootPath)
	filePath = filepath.Clean(filePath)

	fileRelative, err := filepath.Rel(rootPath, filePath)
	if err != nil || escapesRoot(fileRelative) {
		return nil, fmt.Errorf("%s: outside the root", text.Display(file))
	}

	directories := configDirectories(rootPath, filePath)
	files := make([]File, 0, len(directories))
	paths := make([]string, 0, len(directories))
	for _, directory := range directories {
		configPath := filepath.Join(directory, ".editorconfig")
		parsed, _, exists, err := readConfig(rootPath, configPath)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		files = append(files, parsed)
		paths = append(paths, filepath.Dir(configPath))
		if parsed.Root {
			break
		}
	}

	settings := make(map[string]Setting)
	for index := len(files) - 1; index >= 0; index-- {
		target, err := filepath.Rel(paths[index], filePath)
		if err != nil {
			return nil, fmt.Errorf("%s: outside the root", text.Display(file))
		}
		target = filepath.ToSlash(target)
		for _, section := range files[index].Sections {
			if !Match(section.Glob, target) {
				continue
			}
			for _, prop := range section.Props {
				if prop.Value == "unset" {
					delete(settings, prop.Key)
					continue
				}
				settings[prop.Key] = Setting{
					Value: prop.Value,
					File:  files[index].Display,
					Line:  prop.Line,
				}
			}
		}
	}

	return settings, nil
}

func configDirectories(root, file string) []string {
	directory := filepath.Dir(file)
	if samePath(root, file) {
		directory = root
	}

	var directories []string
	for {
		directories = append(directories, directory)
		if samePath(directory, root) {
			break
		}
		parent := filepath.Dir(directory)
		if samePath(parent, directory) {
			break
		}
		directory = parent
	}
	return directories
}

func readConfig(root, path string) (File, string, bool, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		display := text.Display(path)
		return File{}, display, false, fmt.Errorf("%s: cannot read the file", display)
	}
	display := filepath.ToSlash(relative)

	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return File{}, display, false, nil
	}
	if err != nil {
		return File{}, display, false, fmt.Errorf("%s: cannot read the file", display)
	}
	if info.IsDir() {
		return File{}, display, false, fmt.Errorf("%s: cannot read the file", display)
	}

	// A non-regular target cannot be parsed as a config file and reading a
	// special file such as a pipe could block indefinitely.
	info, err = os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return File{}, display, false, fmt.Errorf("%s: cannot read the file", display)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, display, false, fmt.Errorf("%s: cannot read the file", display)
	}

	parsed, err := Parse(display, string(data))
	if err != nil {
		return File{}, display, false, err
	}
	return parsed, display, true, nil
}

func escapesRoot(relative string) bool {
	return filepath.IsAbs(relative) ||
		relative == ".." ||
		strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
}
