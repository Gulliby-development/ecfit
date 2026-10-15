<p align="center"><img src="docs/logo.svg" width="96" alt="" /></p>

# ecfit

ecfit is an offline command-line checker for teams that maintain an
[EditorConfig](https://editorconfig.org/) file. Use it in local development or
CI to see which settings apply to a file, find violations, and safely fix the
violations that can be corrected without changing code structure.

[![CI](https://github.com/Gulliby-development/ecfit/actions/workflows/ci.yml/badge.svg)](https://github.com/Gulliby-development/ecfit/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Install

Build the executable from the repository root:

```sh
go build -o ecfit .
```

You can also run it directly with Go:

```sh
go run . ARGS
```

## Usage

The standalone help output is:

```text
usage: ecfit explain [--root DIR] FILE
       ecfit check [--root DIR] PATH...
       ecfit fix [--root DIR] PATH...
```

Pass `--root DIR` or `--root=DIR` to choose the project root; the default is
the current directory. Repeated root options use the last value. `explain`
does not require the target file to exist. `check` and `fix` accept files or
directories; directories are traversed recursively.

For example, given the configuration and files described below, explaining
`web/app.js` prints:

```text
web/app.js
  charset = utf-8                  .editorconfig:7
  end_of_line = lf                 .editorconfig:4
  indent_size = 4                  web/.editorconfig:2
  indent_style = space             .editorconfig:13
  insert_final_newline = true      .editorconfig:5
  max_line_length = 60             web/.editorconfig:3
  trim_trailing_whitespace = true  .editorconfig:6
```

Checking the project reports:

```text
./data.json:1: EC007 unexpected byte order mark
./docs/guide.md:4: EC001 line ends with crlf, expected lf
./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80
./main.go:4: EC002 trailing whitespace
./util.go:4: EC004 indented with a space, expected tabs
./util.go:5: EC003 missing final newline
./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4
./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4
./web/app.js:4: EC004 indented with a tab, expected spaces
./web/app.js:5: EC006 line is 63 characters long, the limit is 60
10 problems in 5 files, 8 files checked
```

Fixing the project reports the remaining issues:

```text
./data.json: fixed EC007
./docs/guide.md: fixed EC001
./docs/guide.md:4: EC006 line is 87 characters long, the limit is 80
./main.go: fixed EC002
./util.go: fixed EC003
./util.go:4: EC004 indented with a space, expected tabs
./web/app.js:2: EC005 indent of 2 spaces is not a multiple of 4
./web/app.js:3: EC005 indent of 3 spaces is not a multiple of 4
./web/app.js:4: EC004 indented with a tab, expected spaces
./web/app.js:5: EC006 line is 63 characters long, the limit is 60
fixed 4 files, 6 problems left
```

## Settings

ecfit reads applicable `.editorconfig` files from the target's directory up
through the root, applying matching sections from outermost to innermost.
A nearer assignment overrides an earlier one; `unset` removes a setting.
The supported keys are `indent_style`, `indent_size`, `tab_width`,
`end_of_line`, `charset`, `trim_trailing_whitespace`,
`insert_final_newline`, and `max_line_length`.

This is the project configuration used in the examples:

```ini
root = true

[*]
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
charset = utf-8

[*.{go,mod}]
indent_style = tab

[*.{js,json,yml}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab

[docs/**.md]
max_line_length = 80
trim_trailing_whitespace = false
```

## Rules

`check` reports line-ending mismatches (EC001), trailing whitespace (EC002),
final-newline mismatches (EC003), indentation style violations (EC004),
indent-size violations (EC005), maximum line-length violations (EC006), and
UTF-8 byte-order-mark or validity problems (EC007). Files containing a NUL
byte are skipped.

`fix` safely handles byte-order marks, trailing whitespace, line endings, and
final newlines. It leaves indentation, line length, and invalid UTF-8 issues
for you to resolve manually, then reports any findings that remain. It writes
only changed files.

## Globs

Patterns without a slash match basenames at any depth; patterns containing a
slash match the whole path relative to the configuration file. ecfit supports
`*`, `**`, `**/`, `?`, character classes and ranges, nested brace
alternatives, numeric ranges, and backslash escaping. Matching is
case-sensitive.

## Exit codes

- `0`: command completed without remaining problems.
- `1`: `check` found problems, or `fix` left problems after safe fixes.
- `2`: usage or input error. The diagnostic is written to standard error.

## Development

Run the project checks from the repository root:

```sh
go vet ./...
gofmt -l .
go test ./...
```

## License

MIT, see [LICENSE](LICENSE).

---

Built with BAKED
