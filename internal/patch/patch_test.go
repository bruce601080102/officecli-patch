package patch

import (
	"strings"
	"testing"
)

func TestBuildOnlyCreatesTextSet(t *testing.T) {
	original := []byte(`[{"command":"meta","dumpVersion":2},{"command":"add","parent":"/body/p[1]","type":"r","props":{"text":"old","bold":"true"}}]`)
	ai := []byte(`[{"command":"meta","dumpVersion":2},{"command":"add","parent":"/body/p[1]","type":"r","props":{"text":"new","bold":"true"}}]`)
	commands, changes, err := Build(original, ai)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Path != "/body/p[1]/r[1]" || commands[0].Props["text"] != "new" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
	if len(changes) != 1 || changes[0].Old != "old" {
		t.Fatalf("unexpected changes: %#v", changes)
	}
}

func TestBuildIgnoresStyleMutation(t *testing.T) {
	original := []byte(`[{"command":"add","parent":"/body/p[1]","type":"r","props":{"text":"old","bold":"true"}}]`)
	ai := []byte(`[{"command":"add","parent":"/body/p[1]","type":"r","props":{"text":"new","bold":"false"}}]`)
	commands, _, err := Build(original, ai)
	if err != nil || len(commands) != 1 || commands[0].Props["text"] != "new" {
		t.Fatalf("expected only text to be emitted, got %#v, %v", commands, err)
	}
}

func TestBuildCreatesExcelCellPatches(t *testing.T) {
	original := []byte(`[{"command":"meta","dumpVersion":2},{"command":"import","parent":"/Sheet1","props":{"start-cell":"A1"},"text":"Name,Pay\nAlice,10\nTotal,=SUM(B2)\n"}]`)
	ai := []byte(`[{"command":"meta","dumpVersion":2},{"command":"import","parent":"/Sheet1","props":{"start-cell":"A1"},"text":"Name,Pay\nAlice,20\nTotal,=SUM(B2:B3)\n"}]`)
	commands, changes, err := Build(original, ai)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 2 || commands[0].Command != "set" || commands[0].Path != "/Sheet1/B2" || commands[0].Props["value"] != "20" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
	if commands[1].Path != "/Sheet1/B3" || commands[1].Props["formula"] != "SUM(B2:B3)" {
		t.Fatalf("unexpected formula command: %#v", commands[1])
	}
	if len(changes) != 2 || changes[0].Path != "/Sheet1/B2" {
		t.Fatalf("unexpected changes: %#v", changes)
	}
	data, err := Marshal(commands)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"path": "/Sheet1/B2"`) || !strings.Contains(string(data), `"formula": "SUM(B2:B3)"`) {
		t.Fatalf("unexpected patch JSON: %s", data)
	}
}

func TestBuildCreatesExcelClearAtOffsetStartCell(t *testing.T) {
	original := []byte(`[{"command":"import","parent":"/Data","props":{"start-cell":"D5"},"text":"Keep,Remove\n"}]`)
	ai := []byte(`[{"command":"import","parent":"/Data","props":{"start-cell":"D5"},"text":"Keep,\n"}]`)
	commands, _, err := Build(original, ai)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Path != "/Data/E5" || commands[0].Props["clear"] != "true" {
		t.Fatalf("unexpected clear command: %#v", commands)
	}
}

func TestBuildCreatesPowerPointTextPatch(t *testing.T) {
	original := []byte(`[{"command":"set","path":"/slide[1]/shape[1]/paragraph[1]","props":{"text":"Old title","alignment":"center"}}]`)
	ai := []byte(`[{"command":"set","path":"/slide[1]/shape[1]/paragraph[1]","props":{"text":"New title","alignment":"left"}}]`)
	commands, changes, err := Build(original, ai)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Path != "/slide[1]/shape[1]/paragraph[1]" || commands[0].Props["text"] != "New title" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
	if _, included := commands[0].Props["alignment"]; included {
		t.Fatalf("style property leaked into content patch: %#v", commands[0])
	}
	if len(changes) != 1 || changes[0].Old != "Old title" || changes[0].New != "New title" {
		t.Fatalf("unexpected changes: %#v", changes)
	}
}

func TestBuildCreatesExcelRichTextPatch(t *testing.T) {
	original := []byte(`[{"command":"set","path":"/Sheet1/B2","props":{"type":"richtext","runs":"[{\"text\":\"Old\"}]"}}]`)
	ai := []byte(`[{"command":"set","path":"/Sheet1/B2","props":{"type":"richtext","runs":"[{\"text\":\"New\"}]"}}]`)
	commands, _, err := Build(original, ai)
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Props["type"] != "richtext" || commands[0].Props["runs"] != `[{"text":"New"}]` {
		t.Fatalf("unexpected commands: %#v", commands)
	}
}

func TestBuildRejectsShiftedStructure(t *testing.T) {
	original := []byte(`[{"command":"set","path":"/slide[1]/shape[1]/paragraph[1]","props":{"text":"Old"}}]`)
	ai := []byte(`[{"command":"set","path":"/slide[2]/shape[1]/paragraph[1]","props":{"text":"New"}}]`)
	_, _, err := Build(original, ai)
	if err == nil || !strings.Contains(err.Error(), "path differs") {
		t.Fatalf("expected a structural mismatch, got %v", err)
	}
}

func TestColumnName(t *testing.T) {
	for column, want := range map[int]string{1: "A", 26: "Z", 27: "AA", 16384: "XFD"} {
		if got := columnName(column); got != want {
			t.Errorf("columnName(%d) = %q, want %q", column, got, want)
		}
	}
}
