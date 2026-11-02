package model

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type schemaFile struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Defs       map[string]struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	} `json:"$defs"`
}

func TestSchemaFileMatchesStructs(t *testing.T) {
	raw, err := os.ReadFile("../../schema/match.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var s schemaFile
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	check(t, reflect.TypeOf(Result{}), s.Properties)
	check(t, reflect.TypeOf(Player{}), s.Defs["player"].Properties)
	check(t, reflect.TypeOf(SourceEntry{}), s.Defs["source"].Properties)
}

func check(t *testing.T, typ reflect.Type, props map[string]json.RawMessage) {
	t.Helper()
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		if _, ok := props[name]; !ok {
			t.Errorf("%s.%s is missing from the schema", typ.Name(), name)
		}
	}
}
