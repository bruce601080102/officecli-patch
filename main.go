// officecli-patch adds safe JSON text patching to iOfficeAI OfficeCLI.
package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

// rewrite combines the safe portion of the former multi-command workflow:
// copy source -> make a props.text-only patch -> batch the patch -> restore all
// untouched DOCX package parts. It deliberately never replays the full AI dump.
func rewrite(args []string) error {
	fs := newFlagSet("rewrite")
	output := fs.String("output", "", "write rewritten DOCX here")
	fs.StringVar(output, "o", "", "write rewritten DOCX here")
	force := fs.Bool("force", false, "allow overwriting --output")
	bestEffort := fs.Bool("best-effort", false, "continue if an OfficeCLI text path fails")
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
	commands, changes, err := makePatch(fs.Arg(1), fs.Arg(2))
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		fmt.Fprintln(os.Stderr, "officecli-patch: no text changes; output document was not created")
		return nil
	}
	data, err := patch.Marshal(commands)
	if err != nil {
		return fmt.Errorf("serialize patch: %w", err)
	}
	if err := copyDocument(source, *output, *force); err != nil {
		return err
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
	batchArgs := []string{"batch", *output, "--input", patchPath}
	if *bestEffort {
		batchArgs = append(batchArgs, "--best-effort")
	}
	batchLog, batchErr := runBatch(batchArgs)
	closeErr := runOfficeCLI([]string{"close", *output})
	if err := mergeUnmodifiedParts(source, *output, mutablePartsFromPaths(successfulPaths(batchLog))); err != nil {
		return err
	}
	if batchErr != nil {
		return batchErr
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Fprintf(os.Stderr, "officecli-patch: rewrote %d text change(s) to %s\n", len(changes), *output)
	return nil
}

// mutableParts returns the OpenXML package parts which may contain an edited run.
// Every other ZIP part is restored byte-for-byte from the original after OfficeCLI
// saves its text update, preventing its serializer from changing unrelated content.
func mutablePartsFromPaths(paths []string) map[string]bool {
	parts := make(map[string]bool)
	for _, path := range paths {
		root := strings.TrimPrefix(path, "/")
		root = strings.SplitN(root, "/", 2)[0]
		switch {
		case root == "body":
			parts["word/document.xml"] = true
		case root == "styles":
			parts["word/styles.xml"] = true
		case root == "footnotes":
			parts["word/footnotes.xml"] = true
		case root == "endnotes":
			parts["word/endnotes.xml"] = true
		case root == "comments":
			parts["word/comments.xml"] = true
		case strings.HasPrefix(root, "header["):
			if n, ok := bracketNumber(root); ok {
				parts[fmt.Sprintf("word/header%d.xml", n)] = true
			}
		case strings.HasPrefix(root, "footer["):
			if n, ok := bracketNumber(root); ok {
				parts[fmt.Sprintf("word/footer%d.xml", n)] = true
			}
		}
	}
	return parts
}

var updatedPathLine = regexp.MustCompile(`(?m)^\[\d+\] Updated ([^:]+):`)

// successfulPaths is intentionally derived from OfficeCLI's per-item result. A
// failed set must not cause its header/footer part to be retained from a save that
// merely normalized it; it is restored exactly from the source instead.
func successfulPaths(batchLog []byte) []string {
	matches := updatedPathLine.FindAllSubmatch(batchLog, -1)
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		paths = append(paths, string(match[1]))
	}
	return paths
}

func bracketNumber(root string) (int, bool) {
	start := strings.IndexByte(root, '[')
	end := strings.IndexByte(root, ']')
	if start < 0 || end <= start+1 {
		return 0, false
	}
	n, err := strconv.Atoi(root[start+1 : end])
	return n, err == nil && n > 0
}

// mergeUnmodifiedParts takes the selected XML parts from OfficeCLI's result and
// raw-copies all remaining package entries from source. archive/zip.Writer.Copy
// preserves each untouched entry's compressed bytes and metadata.
func mergeUnmodifiedParts(source, edited string, mutable map[string]bool) error {
	originalZip, err := zip.OpenReader(source)
	if err != nil {
		return fmt.Errorf("open original DOCX package: %w", err)
	}
	defer originalZip.Close()
	editedZip, err := zip.OpenReader(edited)
	if err != nil {
		return fmt.Errorf("open edited DOCX package: %w", err)
	}
	defer editedZip.Close()
	editedEntries := make(map[string]*zip.File, len(editedZip.File))
	for _, entry := range editedZip.File {
		editedEntries[entry.Name] = entry
	}

	tmp, err := os.CreateTemp(filepath.Dir(edited), ".officecli-patch-merged-*.docx")
	if err != nil {
		return fmt.Errorf("create merged DOCX: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	writer := zip.NewWriter(tmp)
	for _, originalEntry := range originalZip.File {
		entry := originalEntry
		if mutable[originalEntry.Name] {
			var exists bool
			entry, exists = editedEntries[originalEntry.Name]
			if !exists {
				writer.Close()
				tmp.Close()
				return fmt.Errorf("OfficeCLI result is missing modified part %s", originalEntry.Name)
			}
		}
		if err := writer.Copy(entry); err != nil {
			writer.Close()
			tmp.Close()
			return fmt.Errorf("copy DOCX part %s: %w", entry.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		tmp.Close()
		return fmt.Errorf("finish merged DOCX: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close merged DOCX: %w", err)
	}
	// Zip readers hold Windows file handles until explicitly closed. Release them
	// before replacing the result path (the defers still cover earlier returns).
	if err := originalZip.Close(); err != nil {
		return fmt.Errorf("close original DOCX package: %w", err)
	}
	if err := editedZip.Close(); err != nil {
		return fmt.Errorf("close edited DOCX package: %w", err)
	}
	if err := replaceFile(tmpName, edited); err != nil {
		return fmt.Errorf("replace edited DOCX with preserved package: %w", err)
	}
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

func runBatch(args []string) ([]byte, error) {
	var output bytes.Buffer
	err := runOfficeCLIWithOutput(args, &output)
	return output.Bytes(), err
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
  officecli-patch rewrite <source.docx> <original.json> <ai.json> [-o <edited.docx>] [--force] [--best-effort]

rewrite combines copy, diff, and OfficeCLI batch into one operation. It applies only actual
props.text differences, never replays the full AI JSON, and restores every untouched DOCX ZIP
part from the source document.

All other commands and arguments are passed through to the embedded native OfficeCLI unchanged.
For example: officecli-patch dump file.docx -o original.json
`)
}
