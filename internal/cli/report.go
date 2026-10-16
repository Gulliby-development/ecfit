// Package cli implements ecfit's command-line interface.
package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Gulliby-development/ecfit/internal/config"
	"github.com/Gulliby-development/ecfit/internal/rules"
	"github.com/Gulliby-development/ecfit/internal/text"
)

// FileReport contains the staged output for one expanded file. Findings are
// those that remain after any requested fix; OriginalFindings are used to name
// the fixes that were applied.
type FileReport struct {
	Path             string
	Findings         []rules.Finding
	OriginalFindings []rules.Finding
	Changed          bool
}

// FormatHelp returns the standalone help text with one final LF.
func FormatHelp() string {
	return Usage + "\n"
}

// FormatExplain formats the resolved settings in stable key order.
func FormatExplain(path string, settings map[string]config.Setting) string {
	var output strings.Builder
	fmt.Fprintf(&output, "%s\n", text.Display(path))
	if len(settings) == 0 {
		output.WriteString("  (no settings)\n")
		return output.String()
	}

	keys := make([]string, 0, len(settings))
	width := 0
	for key, setting := range settings {
		keys = append(keys, key)
		if size := len(key) + len(" = ") + len(setting.Value); size > width {
			width = size
		}
	}
	sort.Strings(keys)

	for _, key := range keys {
		setting := settings[key]
		name := key + " = " + setting.Value
		fmt.Fprintf(&output, "  %-*s  %s:%d\n", width, name, text.Display(setting.File), setting.Line)
	}
	return output.String()
}

// FormatCheck formats diagnostics and the checked-file summary.
func FormatCheck(files []FileReport, checked int) string {
	var output strings.Builder
	problems := 0
	problemFiles := 0
	for _, file := range files {
		if len(file.Findings) != 0 {
			problemFiles++
		}
		for _, finding := range file.Findings {
			fmt.Fprintf(&output, "%s:%d: %s %s\n", text.Display(file.Path), finding.Line, finding.Code, finding.Message)
			problems++
		}
	}

	if problems == 0 {
		fmt.Fprintf(&output, "ok, %d %s checked\n", checked, text.Plural(checked, "file", "files"))
		return output.String()
	}
	fmt.Fprintf(&output, "%d %s in %d %s, %d %s checked\n",
		problems, text.Plural(problems, "problem", "problems"),
		problemFiles, text.Plural(problemFiles, "file", "files"),
		checked, text.Plural(checked, "file", "files"))
	return output.String()
}

// FormatFix formats changed-file notices, remaining diagnostics, and totals.
func FormatFix(files []FileReport) string {
	var output strings.Builder
	fixedFiles := 0
	problems := 0
	for _, file := range files {
		display := text.Display(file.Path)
		if file.Changed {
			fixedFiles++
			codes := fixedCodes(file.OriginalFindings)
			fmt.Fprintf(&output, "%s: fixed %s\n", display, strings.Join(codes, ", "))
		}
		for _, finding := range file.Findings {
			fmt.Fprintf(&output, "%s:%d: %s %s\n", display, finding.Line, finding.Code, finding.Message)
			problems++
		}
	}
	fmt.Fprintf(&output, "fixed %d %s, %d %s left\n",
		fixedFiles, text.Plural(fixedFiles, "file", "files"),
		problems, text.Plural(problems, "problem", "problems"))
	return output.String()
}

func appendSkippedSummary(report string, skipped int) string {
	if skipped == 0 {
		return report
	}
	return strings.TrimSuffix(report, "\n") + fmt.Sprintf(", %d binary %s skipped\n", skipped, text.Plural(skipped, "file", "files"))
}

func fixedCodes(findings []rules.Finding) []string {
	seen := make(map[string]bool)
	codes := make([]string, 0, len(findings))
	for _, finding := range findings {
		fixable := finding.Code == "EC001" || finding.Code == "EC002" || finding.Code == "EC003"
		if finding.Code == "EC007" && (finding.Message == "unexpected byte order mark" || finding.Message == "missing byte order mark") {
			fixable = true
		}
		if fixable && !seen[finding.Code] {
			seen[finding.Code] = true
			codes = append(codes, finding.Code)
		}
	}
	sort.Strings(codes)
	return codes
}
