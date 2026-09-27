// Package patch creates deliberately narrow OfficeCLI batch scripts.
package patch

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Change is one text mutation that can be shown to users or serialized for batch.
type Change struct {
	Path string `json:"path"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

// BatchCommand is an OfficeCLI batch command. It intentionally has no other props.
type BatchCommand struct {
	Command string         `json:"command"`
	Path    string         `json:"path"`
	Props   map[string]any `json:"props"`
}

// Build reads only props.text changes on existing Word run additions. All other AI
// differences are intentionally ignored instead of being replayed. The matching
// command count is retained as a guard against an accidental shifted dump.
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

	runCounts := map[string]int{}
	var commands []BatchCommand
	var changes []Change
	for i := range original {
		oldItem, newItem := original[i], ai[i]
		if !isTextRun(oldItem) {
			continue
		}
		parent := oldItem["parent"].(string)
		runCounts[parent]++
		oldText, err := textValue(oldItem)
		if err != nil {
			return nil, nil, fmt.Errorf("command %d original props.text: %w", i, err)
		}
		newText, err := textValue(newItem)
		if err != nil {
			return nil, nil, fmt.Errorf("command %d AI props.text: %w", i, err)
		}
		if oldText == newText {
			continue
		}
		path := fmt.Sprintf("%s/r[%d]", parent, runCounts[parent])
		commands = append(commands, BatchCommand{Command: "set", Path: path, Props: map[string]any{"text": newText}})
		changes = append(changes, Change{Path: path, Old: oldText, New: newText})
	}
	return commands, changes, nil
}

func Marshal(commands []BatchCommand) ([]byte, error) {
	return json.MarshalIndent(commands, "", "  ")
}

func decodeArray(data []byte) ([]map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var value []map[string]any
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("unexpected trailing JSON")
	}
	return value, nil
}

func isTextRun(item map[string]any) bool {
	return item["command"] == "add" && item["type"] == "r" && isNonEmptyString(item["parent"])
}

func isNonEmptyString(value any) bool {
	s, ok := value.(string)
	return ok && s != ""
}

func textValue(item map[string]any) (string, error) {
	props, ok := item["props"].(map[string]any)
	if !ok {
		return "", nil
	}
	value, exists := props["text"]
	if !exists || value == nil {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("must be a string or null")
	}
	return text, nil
}
