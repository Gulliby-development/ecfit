// Package config models parsed EditorConfig files and resolved settings.
package config

// File is a parsed EditorConfig file. Display is the path used to identify the
// file in diagnostics; sections and properties retain their source order.
type File struct {
	Display  string
	Root     bool
	Sections []Section
}

// Section is a glob and the ordered properties declared beneath it.
type Section struct {
	Glob  string
	Props []Prop
}

// Prop is one recognized property assignment and its source line.
type Prop struct {
	Key   string
	Value string
	Line  int
}

// Setting is the winning value for a property and its source location.
type Setting struct {
	Value string
	File  string
	Line  int
}
