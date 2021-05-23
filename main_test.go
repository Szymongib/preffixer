package main

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var expectedFileNames = []string{
	"testdata/file_1.txt", "testdata/file_4.json", "testdata/file_with_prefix",
	"testdata/inner_dir/DONTREADME.md", "testdata/inner_dir/file_2.txt",
	"testdata/inner_dir/inner_inner_dir/file_3.txt", "testdata/inner_dir/inner_inner_dir/ignore_me.json",
}

var originalTestFiles map[string][]byte = map[string][]byte{}

func TestMain(m *testing.M) {
	if err := readOriginalFiles(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(run(m))
}

func run(m *testing.M) int {
	defer resetFiles()
	return m.Run()
}

func resetFiles() {
	for k, v := range originalTestFiles {
		err := os.WriteFile(k, v, os.ModeType)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error resetting file %q: %s", k, err))
		}
	}
}

func readOriginalFiles() error {
	files, err := walkMatch("testdata", "*")
	if err != nil {
		return err
	}

	if len(files) != len(expectedFileNames) {
		return fmt.Errorf("expected to find %d files, found %d", len(expectedFileNames), len(files))
	}

	for i := range files {
		if files[i] != expectedFileNames[i] {
			return fmt.Errorf("file name did not match expected, expected: %s, actual: %s", expectedFileNames[i], files[i])
		}
		originalTestFiles[files[i]], err = ioutil.ReadFile(files[i])
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}
	}

	return nil
}

func TestPreffixer(t *testing.T) {

	for _, testCase := range []struct {
		description string
		extraArgs   []string
		addToPrefix string
	}{
		{
			description: "without line break",
			extraArgs:   []string{},
		},
		{
			description: "with line break",
			extraArgs:   []string{"-e"},
			addToPrefix: "\n",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			t.Run("inject and remove prefixes in all files", func(t *testing.T) {
				defer resetFiles()
				expectedPrefix := fmt.Sprintf("My new shiny prefix%s", testCase.addToPrefix)

				args := append([]string{fmt.Sprintf("--prefix=%s", "My new shiny prefix")}, testCase.extraArgs...)
				cmd, _ := makeInjectCmd(args)
				err := cmd.Execute()
				require.NoError(t, err)
				assertHavePrefix(t, allKeys(originalTestFiles), expectedPrefix, originalTestFiles)

				// Run one more time to make sure prefix is not added again
				cmd, _ = makeInjectCmd(args)
				err = cmd.Execute()
				require.NoError(t, err)

				cmd, _ = makeRemoveCmd(args)
				err = cmd.Execute()
				require.NoError(t, err)
				assertMatchOriginal(t, originalTestFiles)
			})

			t.Run("match pattern with multiline prefix", func(t *testing.T) {
				defer resetFiles()

				affectedFiles := []string{
					"testdata/file_1.txt", "testdata/inner_dir/file_2.txt", "testdata/inner_dir/inner_inner_dir/file_3.txt",
				}

				prefix := `Copyright 2021
Awesome prefix
Much wow
`
				expectedPrefix := fmt.Sprintf("%s%s", prefix, testCase.addToPrefix)
				args := append(
					[]string{"--pattern=*.txt", fmt.Sprintf("--prefix=%s", prefix)},
					testCase.extraArgs...)

				cmd, _ := makeInjectCmd(args)
				err := cmd.Execute()
				require.NoError(t, err)
				assertHavePrefix(t, affectedFiles, expectedPrefix, originalTestFiles)

				changedFiles, err := getChangedFiles(originalTestFiles)
				require.NoError(t, err)
				assert.Equal(t, len(affectedFiles), len(changedFiles))
				for _, file := range changedFiles {
					assert.Contains(t, affectedFiles, file)
				}

				cmd, _ = makeRemoveCmd(args)
				err = cmd.Execute()
				require.NoError(t, err)
				assertMatchOriginal(t, originalTestFiles)
			})

			t.Run("read from file", func(t *testing.T) {
				defer resetFiles()
				prefixFile := "testdata/file_with_prefix"
				prefixBytes, err := ioutil.ReadFile(prefixFile)
				require.NoError(t, err)

				affectedFiles := []string{
					"testdata/file_1.txt", "testdata/inner_dir/file_2.txt", "testdata/inner_dir/inner_inner_dir/file_3.txt",
				}
				prefix := string(prefixBytes)

				expectedPrefix := fmt.Sprintf("%s%s", prefix, testCase.addToPrefix)
				args := append(
					[]string{"--pattern=*.txt", fmt.Sprintf("--prefix-file=%s", prefixFile)},
					testCase.extraArgs...)

				cmd, _ := makeInjectCmd(args)
				err = cmd.Execute()
				require.NoError(t, err)
				assertHavePrefix(t, affectedFiles, expectedPrefix, originalTestFiles)

				changedFiles, err := getChangedFiles(originalTestFiles)
				require.NoError(t, err)
				assert.Equal(t, len(affectedFiles), len(changedFiles))
				for _, file := range changedFiles {
					assert.Contains(t, affectedFiles, file)
				}

				cmd, _ = makeRemoveCmd(args)
				err = cmd.Execute()
				require.NoError(t, err)
				assertMatchOriginal(t, originalTestFiles)
			})
		})
	}
}

func makeInjectCmd(args []string) (*cobra.Command, *bytes.Buffer) {
	injectArgs := []string{"inject", "testdata"}
	cmd, buff := getCmd()
	cmd.SetArgs(append(injectArgs, args...))
	return cmd, buff
}

func makeRemoveCmd(args []string) (*cobra.Command, *bytes.Buffer) {
	injectArgs := []string{"remove", "testdata"}
	cmd, buff := getCmd()
	cmd.SetArgs(append(injectArgs, args...))
	return cmd, buff
}

func getCmd() (*cobra.Command, *bytes.Buffer) {
	rootCmd := rootCommand()

	outBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)

	return rootCmd, outBuf
}

func allKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))

	for k, _ := range m {
		out = append(out, k)
	}
	return out
}

func assertHavePrefix(t *testing.T, files []string, prefix string, original map[string][]byte) {
	for _, f := range files {
		file, err := ioutil.ReadFile(f)
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(string(file), prefix))
		noPref := strings.TrimPrefix(string(file), prefix)
		assert.Equal(t, original[f], []byte(noPref))
	}
}

func assertMatchOriginal(t *testing.T, original map[string][]byte) {
	for p, v := range original {
		content, err := ioutil.ReadFile(p)
		require.NoError(t, err)

		assert.Equal(t, v, content)
	}
}

func getChangedFiles(original map[string][]byte) ([]string, error) {
	out := make([]string, 0)
	for p, v := range original {
		content, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}

		if string(content) != string(v) {
			out = append(out, p)
		}
	}
	return out, nil
}
