# Prefix fixer

Preffixer is a tool that allows you to quickly add and remove prefixes from file contents walking down the specified path.

The prefix is not injected if file already starts with the provided prefix.

This tool might be useful for manipulating Go build tags or adding boilerplate/licence notes to files in your project.

## Example

Given the following directory structure:
```
|-- cmd
|-- pkg
|-- e2e
|   |-- README.md
|   |-- run_tests.sh
|   |-- tests
|   |   |-- awesome_test.go
|   |   |-- better_test.go
|   |   |-- flaky_test.go
|   |-- testutil
|   |   |-- do-something.go
|   |   |-- whatever.go
```
You can easily add build tag `e2e` to all your `.go` files in `e2e` directory:
```bash
preffixer inject ./e2e --prefix="//+build e2e" --pattern "*.go" -e
```

## Installation

Install with `go get`:
```bash
GO111MODULE=off go get github.com/Szymongib/preffixer
```

## Usage

```
Add or remove prefixes from all files matching the pattern in directory.

Usage:
  preffixer [flags]
  preffixer [command]

Available Commands:
  help        Help about any command
  inject      Inject prefix to all files down the root path matching the pattern, that does not already start with it.
  remove      Remove prefix from all files down the root path matching the pattern.

Flags:
  -h, --help   help for preffixer

Use "preffixer [command] --help" for more information about a command.
```

### Additional flags

- Use `-e` add/remove line break during injection/removal.
- Use `--prefix-file [FILE_PATH]` to read prefix from file.
