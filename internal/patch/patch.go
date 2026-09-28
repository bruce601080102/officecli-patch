// Package patch creates deliberately narrow OfficeCLI batch scripts.
package patch

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Change is one content mutation that can be shown to users or serialized for batch.
type Change struct {
	Path string `json:"path"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// BatchCommand is the subset of an OfficeCLI batch command used by a content patch.
type BatchCommand struct {
	Command string                 `json:"command"`
	Path    string                 `json:"path,omitempty"`
	Props   map[string]interface{} `json:"props,omitempty"`
}

// Build extracts content-only changes from replayable OfficeCLI dumps:
//   - DOCX run text (add ... type=r, props.text)
//   - PPTX text at a stable set path (props.text)
//   - XLSX CSV/TSV import payloads and rich-text cell runs
//
// Formatting and structural differences are deliberately ignored. Command
// count and structural identities are retained as guards against comparing a
// shifted or unrelated dump.
func Build(originalJSON, aiJSON []byte) ([]BatchCommand, []Change, error) {
	original, err := decodeArray(originalJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("read original JSON: %w", err)
	}
	ai, err := decodeArray(aiJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("read AI JSON: %w", err)
	}
	if len(original) != len(ai) {
		return nil, nil, fmt.Errorf("JSON command count differs (original %d, AI %d); refusing structural patch", len(original), len(ai))
	}

	addCounts := map[string]int{}
	var commands []BatchCommand
	var changes []Change
	for i := range original {
		oldItem, newItem := original[i], ai[i]
		if err := sameStructure(oldItem, newItem); err != nil {
			return nil, nil, fmt.Errorf("command %d: %w", i, err)
		}

		if isTextRunAdd(oldItem) {
			parent := oldItem["parent"].(string)
			typ := oldItem["type"].(string)
			countKey := parent + "\x00" + typ
			addCounts[countKey]++

			oldText, err := stringProp(oldItem, "text")
			if err != nil {
				return nil, nil, fmt.Errorf("command %d original props.text: %w", i, err)
			}
			newText, err := stringProp(newItem, "text")
			if err != nil {
				return nil, nil, fmt.Errorf("command %d AI props.text: %w", i, err)
			}
			if oldText != newText {
				path := indexedChildPath(parent, typ, addCounts[countKey])
				commands = append(commands, BatchCommand{Command: "set", Path: path, Props: map[string]interface{}{"text": newText}})
				changes = append(changes, Change{Path: path, Old: oldText, New: newText})
			}
			continue
		}

		if oldItem["command"] == "import" {
			itemCommands, itemChanges, err := importChanges(oldItem, newItem)
			if err != nil {
				return nil, nil, fmt.Errorf("command %d: %w", i, err)
			}
			commands = append(commands, itemCommands...)
			changes = append(changes, itemChanges...)
			continue
		}

		if oldItem["command"] == "set" && isNonEmptyString(oldItem["path"]) {
			command, itemChanges, changed, err := setContentChange(oldItem, newItem)
			if err != nil {
				return nil, nil, fmt.Errorf("command %d: %w", i, err)
			}
			if changed {
				commands = append(commands, command)
				changes = append(changes, itemChanges...)
			}
		}
	}
	return commands, changes, nil
}

func Marshal(commands []BatchCommand) ([]byte, error) {
	return json.MarshalIndent(commands, "", "  ")
}

func decodeArray(data []byte) ([]map[string]interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var value []map[string]interface{}
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}
	var trailing interface{}
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing JSON")
		}
		return nil, fmt.Errorf("unexpected trailing JSON: %w", err)
	}
	return value, nil
}

func sameStructure(oldItem, newItem map[string]interface{}) error {
	for _, key := range []string{"command", "dumpVersion", "path", "parent", "type", "part", "xpath", "action"} {
		if !reflect.DeepEqual(oldItem[key], newItem[key]) {
			return fmt.Errorf("%s differs; refusing structural patch", key)
		}
	}
	return nil
}

func isTextRunAdd(item map[string]interface{}) bool {
	if item["command"] != "add" || !isNonEmptyString(item["parent"]) {
		return false
	}
	typ, ok := item["type"].(string)
	return ok && (typ == "r" || typ == "run")
}

func importChanges(oldItem, newItem map[string]interface{}) ([]BatchCommand, []Change, error) {
	if !reflect.DeepEqual(oldItem["props"], newItem["props"]) {
		return nil, nil, fmt.Errorf("import options differ; refusing structural patch")
	}
	oldText, err := topLevelString(oldItem, "text")
	if err != nil {
		return nil, nil, fmt.Errorf("original import text: %w", err)
	}
	newText, err := topLevelString(newItem, "text")
	if err != nil {
		return nil, nil, fmt.Errorf("AI import text: %w", err)
	}
	if oldText == newText {
		return nil, nil, nil
	}
	parent, _ := oldItem["parent"].(string)
	if parent == "" {
		return nil, nil, fmt.Errorf("import parent must be a non-empty string")
	}
	props := cloneProps(oldItem)
	oldRows, err := parseDelimitedText(oldText, props)
	if err != nil {
		return nil, nil, fmt.Errorf("parse original import text: %w", err)
	}
	newRows, err := parseDelimitedText(newText, props)
	if err != nil {
		return nil, nil, fmt.Errorf("parse AI import text: %w", err)
	}
	start := "A1"
	if value, ok := props["start-cell"].(string); ok && value != "" {
		start = value
	}
	startColumn, startRow, err := parseCellReference(start)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid import start-cell %q: %w", start, err)
	}

	rowCount := max(len(oldRows), len(newRows))
	var commands []BatchCommand
	var changes []Change
	for rowOffset := 0; rowOffset < rowCount; rowOffset++ {
		columnCount := max(rowLength(oldRows, rowOffset), rowLength(newRows, rowOffset))
		for columnOffset := 0; columnOffset < columnCount; columnOffset++ {
			oldValue := tableValue(oldRows, rowOffset, columnOffset)
			newValue := tableValue(newRows, rowOffset, columnOffset)
			if oldValue == newValue {
				continue
			}
			path := fmt.Sprintf("%s/%s%d", strings.TrimRight(parent, "/"), columnName(startColumn+columnOffset), startRow+rowOffset)
			commands = append(commands, BatchCommand{Command: "set", Path: path, Props: cellContentProps(newValue)})
			changes = append(changes, Change{Path: path, Old: oldValue, New: newValue})
		}
	}
	return commands, changes, nil
}

func parseDelimitedText(text string, props map[string]interface{}) ([][]string, error) {
	delimiter := ','
	if format, ok := props["format"].(string); ok && strings.EqualFold(format, "tsv") {
		delimiter = '\t'
	}
	if value, ok := props["delimiter"].(string); ok && value != "" {
		switch value {
		case `\t`, "tab":
			delimiter = '\t'
		default:
			r, size := utf8.DecodeRuneInString(value)
			if r == utf8.RuneError || size != len(value) {
				return nil, fmt.Errorf("delimiter must be one character")
			}
			delimiter = r
		}
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

func parseCellReference(value string) (int, int, error) {
	value = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "$", ""))
	separator := 0
	for separator < len(value) && value[separator] >= 'A' && value[separator] <= 'Z' {
		separator++
	}
	if separator == 0 || separator == len(value) {
		return 0, 0, fmt.Errorf("must use A1 notation")
	}
	column := 0
	for _, letter := range value[:separator] {
		column = column*26 + int(letter-'A') + 1
	}
	row, err := strconv.Atoi(value[separator:])
	if err != nil || row < 1 || column < 1 {
		return 0, 0, fmt.Errorf("must use A1 notation")
	}
	return column, row, nil
}

func columnName(column int) string {
	var reversed []byte
	for column > 0 {
		column--
		reversed = append(reversed, byte('A'+column%26))
		column /= 26
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return string(reversed)
}

func rowLength(rows [][]string, row int) int {
	if row < 0 || row >= len(rows) {
		return 0
	}
	return len(rows[row])
}

func tableValue(rows [][]string, row, column int) string {
	if row < 0 || row >= len(rows) || column < 0 || column >= len(rows[row]) {
		return ""
	}
	return rows[row][column]
}

func cellContentProps(value string) map[string]interface{} {
	if value == "" {
		return map[string]interface{}{"clear": "true"}
	}
	if strings.HasPrefix(value, "=") {
		return map[string]interface{}{"formula": strings.TrimPrefix(value, "=")}
	}
	return map[string]interface{}{"value": value}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func setContentChange(oldItem, newItem map[string]interface{}) (BatchCommand, []Change, bool, error) {
	path := oldItem["path"].(string)
	props := map[string]interface{}{}
	var changes []Change
	for _, key := range []string{"text", "runs"} {
		oldValue, oldExists, err := optionalStringProp(oldItem, key)
		if err != nil {
			return BatchCommand{}, nil, false, fmt.Errorf("original props.%s: %w", key, err)
		}
		newValue, newExists, err := optionalStringProp(newItem, key)
		if err != nil {
			return BatchCommand{}, nil, false, fmt.Errorf("AI props.%s: %w", key, err)
		}
		if (!oldExists && !newExists) || (oldExists == newExists && oldValue == newValue) {
			continue
		}
		props[key] = newValue
		changes = append(changes, Change{Path: path, Old: oldValue, New: newValue})
		if key == "runs" {
			props["type"] = "richtext"
		}
	}
	if len(props) == 0 {
		return BatchCommand{}, nil, false, nil
	}
	return BatchCommand{Command: "set", Path: path, Props: props}, changes, true, nil
}

func indexedChildPath(parent, typ string, index int) string {
	return strings.TrimRight(parent, "/") + fmt.Sprintf("/%s[%d]", typ, index)
}

func isNonEmptyString(value interface{}) bool {
	s, ok := value.(string)
	return ok && s != ""
}

func stringProp(item map[string]interface{}, key string) (string, error) {
	value, _, err := optionalStringProp(item, key)
	return value, err
}

func optionalStringProp(item map[string]interface{}, key string) (string, bool, error) {
	props, ok := item["props"].(map[string]interface{})
	if !ok {
		return "", false, nil
	}
	value, exists := props[key]
	if !exists || value == nil {
		return "", exists, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", true, fmt.Errorf("must be a string or null")
	}
	return text, true, nil
}

func topLevelString(item map[string]interface{}, key string) (string, error) {
	value, exists := item[key]
	if !exists || value == nil {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("must be a string or null")
	}
	return text, nil
}

func cloneProps(item map[string]interface{}) map[string]interface{} {
	original, _ := item["props"].(map[string]interface{})
	if len(original) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(original))
	for key, value := range original {
		cloned[key] = value
	}
	return cloned
}
