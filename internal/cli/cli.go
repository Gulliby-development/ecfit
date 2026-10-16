// Package cli implements ecfit's command-line interface.
package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Gulliby-development/ecfit/internal/config"
	"github.com/Gulliby-development/ecfit/internal/rules"
	"github.com/Gulliby-development/ecfit/internal/text"
)

// commitWriteFile is the narrow write seam used to exercise commit recovery.
var commitWriteFile = os.WriteFile

type stagedFile struct {
	path     Path
	original []byte
	updated  []byte
	mode     os.FileMode
}

// Run parses and executes one ecfit command. On input failures it writes only a
// prefixed diagnostic to stderr and returns 2.
func Run(args []string, stdout, stderr io.Writer) int {
	parsed, err := ParseArgs(args)
	if err != nil {
		writeError(stderr, err)
		return 2
	}
	if parsed.Help {
		_, _ = io.WriteString(stdout, FormatHelp())
		return 0
	}

	if parsed.Command == "explain" {
		settings, err := config.Resolve(parsed.Root, parsed.Paths[0])
		if err != nil {
			writeError(stderr, err)
			return 2
		}
		_, _ = io.WriteString(stdout, FormatExplain(parsed.Paths[0], settings))
		return 0
	}

	paths, err := ExpandPaths(parsed.Paths)
	if err != nil {
		writeError(stderr, err)
		return 2
	}

	reports := make([]FileReport, 0, len(paths))
	staged := make([]stagedFile, 0, len(paths))
	checked := 0
	skipped := 0
	for _, path := range paths {
		settings, err := config.Resolve(parsed.Root, path.Actual)
		if err != nil {
			writeError(stderr, err)
			return 2
		}
		info, err := os.Stat(path.Actual)
		if err != nil || !info.Mode().IsRegular() {
			writeError(stderr, fmt.Errorf("%s: cannot read the file", text.Display(path.Display)))
			return 2
		}
		data, err := os.ReadFile(path.Actual)
		if err != nil {
			writeError(stderr, fmt.Errorf("%s: cannot read the file", text.Display(path.Display)))
			return 2
		}
		if bytes.IndexByte(data, 0) >= 0 {
			skipped++
			continue
		}

		checked++
		report := FileReport{Path: path.Display}
		if parsed.Command == "check" {
			report.Findings = rules.Check(data, settings)
			reports = append(reports, report)
			continue
		}

		report.OriginalFindings = rules.Check(data, settings)
		updated := rules.Fix(data, settings)
		report.Changed = !bytes.Equal(data, updated)
		report.Findings = rules.Check(updated, settings)
		reports = append(reports, report)
		if report.Changed {
			staged = append(staged, stagedFile{
				path:     path,
				original: data,
				updated:  updated,
				mode:     info.Mode().Perm(),
			})
		}
	}

	if parsed.Command == "fix" {
		if err := commit(staged); err != nil {
			writeError(stderr, err)
			return 2
		}
		_, _ = io.WriteString(stdout, appendSkippedSummary(FormatFix(reports), skipped))
		if hasFindings(reports) {
			return 1
		}
		return 0
	}

	_, _ = io.WriteString(stdout, appendSkippedSummary(FormatCheck(reports, checked), skipped))
	if hasFindings(reports) {
		return 1
	}
	return 0
}

func writeError(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "ecfit: %s\n", err.Error())
}

func hasFindings(reports []FileReport) bool {
	for _, report := range reports {
		if len(report.Findings) != 0 {
			return true
		}
	}
	return false
}

// commit writes only staged changes. If any write fails, the failed target and
// every earlier target are restored from their retained original bytes/modes.
func commit(files []stagedFile) error {
	for index, file := range files {
		if err := commitWriteFile(file.path.Actual, file.updated, file.mode); err != nil {
			rollbackErrors := make([]string, 0)
			for restore := index; restore >= 0; restore-- {
				candidate := files[restore]
				if restoreErr := os.WriteFile(candidate.path.Actual, candidate.original, candidate.mode); restoreErr != nil {
					rollbackErrors = append(rollbackErrors, fmt.Sprintf("%s: %v", text.Display(candidate.path.Display), restoreErr))
				}
			}
			if len(rollbackErrors) != 0 {
				return fmt.Errorf("%s: cannot write the file (rollback failed: %s)", text.Display(file.path.Display), strings.Join(rollbackErrors, "; "))
			}
			return fmt.Errorf("%s: cannot write the file", text.Display(file.path.Display))
		}
	}
	return nil
}
