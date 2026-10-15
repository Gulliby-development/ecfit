// Package cli implements ecfit's command-line interface.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Gulliby-development/ecfit/internal/text"
)

// Usage is the complete help text, without its final line feed.
const Usage = "usage: ecfit explain [--root DIR] FILE\n" +
	"       ecfit check [--root DIR] PATH...\n" +
	"       ecfit fix [--root DIR] PATH..."

// Args contains the validated command and its operands.
type Args struct {
	Command string
	Root    string
	Paths   []string
	Help    bool
}

// ParseArgs parses command-line arguments and validates the selected root.
// Argument errors are returned before the root is checked.
func ParseArgs(argv []string) (Args, error) {
	if len(argv) == 1 && argv[0] == "--help" {
		return Args{Help: true}, nil
	}
	if len(argv) == 0 {
		return Args{}, errors.New("missing command (explain, check or fix)")
	}

	command := argv[0]
	if command != "explain" && command != "check" && command != "fix" {
		return Args{}, fmt.Errorf("unknown command %s", text.Quote(command))
	}

	root := "."
	paths := make([]string, 0, len(argv)-1)
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "--root":
			if i+1 == len(argv) {
				return Args{}, errors.New("--root needs a value")
			}
			i++
			root = argv[i]
		case strings.HasPrefix(arg, "--root="):
			root = strings.TrimPrefix(arg, "--root=")
		case strings.HasPrefix(arg, "-"):
			return Args{}, fmt.Errorf("unknown option %s", text.Quote(arg))
		default:
			paths = append(paths, arg)
		}
	}

	if len(paths) == 0 {
		if command == "explain" {
			return Args{}, errors.New("missing FILE")
		}
		return Args{}, errors.New("missing PATH")
	}
	if command == "explain" && len(paths) > 1 {
		return Args{}, errors.New("too many arguments")
	}

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Args{}, fmt.Errorf("%s: not a directory", text.Display(root))
	}
	return Args{Command: command, Root: root, Paths: paths}, nil
}
