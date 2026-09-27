// officecli-patch adds safe JSON text patching to iOfficeAI OfficeCLI.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"officecli-patch/internal/patch"
)

const version = "0.2.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, "officecli-patch:", err)
		os.Exit(1)
	}
}

type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

func run(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || (args[0] == "help" && len(args) == 1) {
		printUsage(os.Stdout)
		return nil
	}
	if args[0] == "--version" {
		fmt.Println("officecli-patch", version)
		return nil
	}
	switch args[0] {
	case "diff":
		return diff(args[1:])
	case "rewrite":
		return rewrite(args[1:])
	default:
		return runOfficeCLI(args)
	}
}

func diff(args []string) error {
	fs := newFlagSet("diff")
	output := fs.String("output", "", "write OfficeCLI batch patch JSON")
	fs.StringVar(output, "o", "", "write OfficeCLI batch patch JSON")
	if err := parseInterspersed(fs, args, map[string]bool{"--output": true, "-o": true}); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return usageError("diff requires <original.json> <ai.json>")
	}
	commands, changes, err := makePatch(fs.Arg(0), fs.Arg(1))
	if err != nil {
		return err
	}
	data, err := patch.Marshal(commands)
	if err != nil {
		return fmt.Errorf("serialize patch: %w", err)
	}
	if *output == "" || *output == "-" {
		_, err = os.Stdout.Write(append(data, '\n'))
	} else {
		err = writeNewFile(*output, data)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "officecli-patch: %d text change(s)\n", len(changes))
	return nil
}

// rewrite is intentionally equivalent to the proven manual workflow:
// copy source -> batch the full AI JSON -> make a text patch -> batch the patch
// with --best-effort. It does not inspect or rewrite DOCX ZIP parts itself.
func rewrite(args []string) error {
	fs := newFlagSet("rewrite")
	output := fs.String("output", "", "write rewritten DOCX here")
	fs.StringVar(output, "o", "", "write rewritten DOCX here")
	force := fs.Bool("force", false, "allow overwriting --output")
	if err := parseInterspersed(fs, args, map[string]bool{"--output": true, "-o": true}); err != nil {
		return err
	}
	if fs.NArg() != 3 {
		return usageError("rewrite requires <source.docx> <original.json> <ai.json>")
	}
	source := fs.Arg(0)
	if *output == "" {
		*output = strings.TrimSuffix(source, filepath.Ext(source)) + ".rewritten.docx"
	}
	if samePath(source, *output) {
		return usageError("--output is the input file; choose another output name")
	}
	if err := copyDocument(source, *output, *force); err != nil {
		return err
	}
	// This is the first OfficeCLI batch from the manual workflow. It deliberately
	// receives the complete AI JSON rather than the generated narrow text patch.
	if err := runOfficeCLI([]string{"batch", *output, "--input", fs.Arg(2)}); err != nil {
		var exitErr *exitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("apply AI JSON to copied document: %w", err)
		}
		// Match the shell workflow: an OfficeCLI batch may report unresolved items
		// (and even atomic rollback) but the following best-effort text patch must
		// still run against the copied source document.
		fmt.Fprintln(os.Stderr, "officecli-patch: AI JSON batch reported unresolved items; continuing with the text patch")
	}
	commands, changes, err := makePatch(fs.Arg(1), fs.Arg(2))
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		fmt.Fprintf(os.Stderr, "officecli-patch: AI JSON applied; no props.text patch was needed: %s\n", *output)
		return nil
	}
	data, err := patch.Marshal(commands)
	if err != nil {
		return fmt.Errorf("serialize patch: %w", err)
	}
	patchFile, err := os.CreateTemp(filepath.Dir(*output), ".officecli-patch-*.json")
	if err != nil {
		return fmt.Errorf("create temporary patch: %w", err)
	}
	patchPath := patchFile.Name()
	defer os.Remove(patchPath)
	if _, err := patchFile.Write(data); err != nil {
		patchFile.Close()
		return fmt.Errorf("write temporary patch: %w", err)
	}
	if err := patchFile.Close(); err != nil {
		return fmt.Errorf("finish temporary patch: %w", err)
	}
	// This is the second batch from the manual workflow. OfficeCLI can return a
	// non-zero status for individual unresolved paths even after applying the
	// remaining changes, so preserve the written output and report that condition.
	if err := runOfficeCLI([]string{"batch", *output, "--input", patchPath, "--best-effort"}); err != nil {
		var exitErr *exitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("apply text patch: %w", err)
		}
		fmt.Fprintf(os.Stderr, "officecli-patch: text patch completed with some unresolved paths; output was written to %s\n", *output)
	}
	fmt.Fprintf(os.Stderr, "officecli-patch: completed AI JSON batch and %d props.text patch change(s) for %s\n", len(changes), *output)
	return nil
}

func makePatch(originalPath, aiPath string) ([]patch.BatchCommand, []patch.Change, error) {
	original, err := os.ReadFile(originalPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read original JSON: %w", err)
	}
	ai, err := os.ReadFile(aiPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read AI JSON: %w", err)
	}
	return patch.Build(original, ai)
}

func runOfficeCLI(args []string) error {
	return runOfficeCLIWithOutput(args, nil)
}

func runOfficeCLIWithOutput(args []string, capture *bytes.Buffer) error {
	binary, err := embeddedOfficeCLIPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if capture != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, capture)
		cmd.Stderr = io.MultiWriter(os.Stderr, capture)
	}
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return &exitError{code: exit.ExitCode()}
		}
		return fmt.Errorf("run embedded OfficeCLI: %w", err)
	}
	return nil
}

func copyDocument(source, target string, force bool) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source document: %w", err)
	}
	defer in.Close()
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	out, err := os.OpenFile(target, flags, 0600)
	if err != nil {
		return fmt.Errorf("create output document: %w", err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("copy document: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("finish copied document: %w", closeErr)
	}
	return nil
}

func writeNewFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("finish %s: %w", path, err)
	}
	return nil
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	return errA == nil && errB == nil && strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// parseInterspersed accepts native-CLI-style options on either side of positional
// arguments. The standard library flag package otherwise stops at the first path.
func parseInterspersed(fs *flag.FlagSet, args []string, valueFlags map[string]bool) error {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		name := arg
		if equals := strings.IndexByte(arg, '='); equals >= 0 {
			name = arg[:equals]
		}
		if strings.HasPrefix(name, "-") {
			flags = append(flags, arg)
			if valueFlags[name] && !strings.Contains(arg, "=") {
				if i+1 >= len(args) {
					return fmt.Errorf("flag needs a value: %s", name)
				}
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, arg)
	}
	return fs.Parse(append(flags, positional...))
}

func usageError(message string) error {
	return fmt.Errorf("%s (run 'officecli-patch help')", message)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `officecli-patch: safe DOCX text rewrite tool

Usage:
  officecli-patch diff <original.json> <ai.json> [-o <patch.json>]
  officecli-patch rewrite <source.docx> <original.json> <ai.json> [-o <edited.docx>] [--force]

rewrite combines the proven four-command workflow into one operation: copy the source, run the
complete AI JSON through OfficeCLI batch, create a props.text patch, then apply that patch with
OfficeCLI batch --best-effort.

All other commands and arguments are passed through to the embedded native OfficeCLI unchanged.
For example: officecli-patch dump file.docx -o original.json
`)
}
