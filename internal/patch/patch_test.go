package patch

import "testing"

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
