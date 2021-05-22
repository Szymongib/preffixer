package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "preffixer",
		Short: "Quickly manipulate files content prefixes.",
		Long:  `Add or remove prefixes from all files matching the pattern in directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	rootCmd.AddCommand(injectCommand())
	rootCmd.AddCommand(removeCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type opts struct {
	rootPath    string
	prefix      string
	pattern     string
	withLineEnd bool
}

func optsFlags(cmd *cobra.Command) {
	cmd.Flags().String("pattern", "*", "File pattern specifying files to modify.")
	cmd.Flags().String("prefix", "", "Prefix to inject or remove")
	cmd.Flags().String("prefix-file", "", "File from which prefix to inject or remove should be read.")
	cmd.Flags().BoolP("with-line-end", "e", false, "Instructs app to additionally add/remove line break after prefix.")
}

func parseOpts(cmd *cobra.Command, args []string) (opts, error) {
	if len(args) < 1 {
		return opts{}, fmt.Errorf("requires 1 argument [ROOT_PATH]")
	}

	path := args[0]
	if path == "" {
		return opts{}, fmt.Errorf("requires 1 argument [ROOT_PATH]")
	}

	prefix, _ := cmd.Flags().GetString("prefix")
	if prefix == "" {
		prefixFile, _ := cmd.Flags().GetString("prefix-file")
		var err error
		prefix, err = loadFile(prefixFile)
		if err != nil {
			return opts{}, errors.Wrap(err, "failed to load content of prefix file")
		}
	}
	if prefix == "" {
		return opts{}, fmt.Errorf("prefix not provided, specify --prefix or --prefix-file")
	}

	pattern, _ := cmd.Flags().GetString("pattern")
	withLineEnd, _ := cmd.Flags().GetBool("with-line-end")

	return opts{
		rootPath:    path,
		prefix:      prefix,
		pattern:     pattern,
		withLineEnd: withLineEnd,
	}, nil
}

func injectCommand() *cobra.Command {
	newCmd := &cobra.Command{
		Use:     "inject",
		Short:   "Inject prefix to all files down the root path matching the pattern, that does not already start with it.",
		Example: `preffixer inject ./e2e-tests --prefix "//+build e2e" --pattern *.go"`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseOpts(cmd, args)
			if err != nil {
				return err
			}
			return injectCmd(opts)
		},
	}
	optsFlags(newCmd)
	return newCmd
}

func removeCommand() *cobra.Command {
	newCmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "Remove prefix from all files down the root path matching the pattern.",
		Example: `preffixer remove ./e2e-tests --prefix "//+build e2e" --pattern *.go"`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseOpts(cmd, args)
			if err != nil {
				return err
			}
			return removeCmd(opts)
		},
	}
	optsFlags(newCmd)
	return newCmd
}

func injectCmd(options opts) error {
	fmt.Println("Prefix ", options.prefix)
	fmt.Println("Pattern ", options.pattern)

	files, err := getFilePaths(options.rootPath, options.pattern)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Starting injection")
	fmt.Println()

	for _, f := range files {
		injected, err := injectPrefix(f, options.prefix, options.withLineEnd)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error injecting prefix to file %s: %s", f, err))
		}
		if !injected {
			fmt.Println(fmt.Sprintf("File %s already has the prefix", f))
		}
	}

	fmt.Println()
	fmt.Println("Injection finished")
	return nil
}

func removeCmd(options opts) error {
	files, err := getFilePaths(options.rootPath, options.pattern)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Starting removal")
	fmt.Println()

	for _, f := range files {
		removed, err := removePrefix(f, options.prefix, options.withLineEnd)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error removing prefix from file %s: %s", f, err))
		}
		if !removed {
			fmt.Println(fmt.Sprintf("File %s did not have the prefix", f))
		}
	}

	fmt.Println()
	fmt.Println("Removal finished")
	return nil
}

func getFilePaths(rootPath, pattern string) ([]string, error) {
	files, err := walkMatch(rootPath, pattern)
	if err != nil {
		return nil, errors.Wrap(err, "error walking root path")
	}

	if len(files) == 0 {
		fmt.Println("No files matching the pattern found")
		return []string{}, nil
	}

	fmt.Println(fmt.Sprintf("Found %d files matching the pattern: ", len(files)))
	for _, f := range files {
		fmt.Println(fmt.Sprintf("  • %s", f))
	}

	return files, nil
}

func walkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func injectPrefix(path string, prefix string, lineEnd bool) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(string(content), prefix) {
		return false, nil
	}

	newContent := []byte(prefix)
	if lineEnd {
		newContent = append(newContent, '\n')
	}
	newContent = append(newContent, content...)

	err = os.WriteFile(path, newContent, os.ModeType)
	if err != nil {
		return false, err
	}

	return true, nil
}

func removePrefix(path string, prefix string, lineEnd bool) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	if !strings.HasPrefix(string(content), prefix) {
		return false, nil
	}
	newStr := strings.TrimPrefix(string(content), prefix)
	if lineEnd {
		newStr = strings.TrimPrefix(newStr, "\n")
	}

	err = os.WriteFile(path, []byte(newStr), os.ModeType)
	if err != nil {
		return false, err
	}

	return true, nil
}

func loadFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file content")
	}
	return string(content), nil
}
