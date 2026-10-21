package app

import (
	"testing"
)

func TestParseType(t *testing.T) {
	ok := []string{"bool", "uint8", "uint64", "int32", "usize", "float", "float32", "rune", "string", "bytes", "action:Action", "enum:Level"}
	for _, s := range ok {
		if _, err := parseType(s); err != nil {
			t.Errorf("parseType(%q): %v", s, err)
		}
	}
	for _, s := range []string{"", "nat", "array:uint8", "action:", "String"} {
		if _, err := parseType(s); err == nil {
			t.Errorf("parseType(%q) unexpectedly succeeded", s)
		}
	}
}

func TestTypeMappings(t *testing.T) {
	cases := []struct {
		token string
		c     string
		goT   string
	}{
		{"bool", "uint8_t", "bool"},
		{"uint8", "uint8_t", "uint8"},
		{"uint64", "uint64_t", "uint64"},
		{"usize", "size_t", "uint"},
		{"float", "double", "float64"},
		{"rune", "uint32_t", "rune"},
		{"string", "lean_object*", "string"},
		{"bytes", "lean_object*", "[]byte"},
		{"action:Action", "uint8_t", "Action"},
	}
	for _, tc := range cases {
		typ, err := parseType(tc.token)
		if err != nil {
			t.Fatal(err)
		}
		if got := typ.CType(); got != tc.c {
			t.Errorf("%s CType = %q, want %q", tc.token, got, tc.c)
		}
		if got := typ.GoType(nil, Binding{}); got != tc.goT {
			t.Errorf("%s GoType = %q, want %q", tc.token, got, tc.goT)
		}
	}
}

func TestConfigRoundTripWithActions(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Schema:  1,
		Project: Project{Name: "demo", LeanDir: "lean", GoModule: "example.com/demo", PackageDir: "internal/verified", Targets: []string{"linux/amd64"}},
		Lean:    LeanCfg{Toolchain: "4.29.1"},
		Bindings: []Binding{{
			Package: "authz",
			Module:  "Authz",
			Export:  "leanport_authz_authorize",
			GoName:  "Authorize",
			Args:    []string{"bool", "action:Action"},
			Result:  "bool",
			Runtime: "auto",
			Actions: map[string]uint8{"Create": 0, "Read": 1},
		}},
	}
	if err := saveConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Bindings) != 1 || len(got.Bindings[0].Actions) != 2 {
		t.Fatalf("actions did not round-trip: %+v", got.Bindings)
	}
	if got.Bindings[0].Actions["Read"] != 1 {
		t.Fatalf("wrong action code: %+v", got.Bindings[0].Actions)
	}
}

func TestChooseMode(t *testing.T) {
	scalar := Binding{Args: []string{"bool", "uint8"}, Result: "bool", Runtime: "auto"}
	mode, err := chooseMode(scalar, []string{"initialize_Init", "lean_inc_heartbeat", "mi_malloc_small", "memcpy"}, "")
	if err != nil || mode != "none" {
		t.Fatalf("scalar mode = %q, %v; want none", mode, err)
	}
	linked, err := chooseMode(scalar, []string{"lean_alloc_ctor"}, "")
	if err != nil || linked != "linked" {
		t.Fatalf("live runtime symbol should force linked, got %q, %v", linked, err)
	}
	str := Binding{Args: []string{"string"}, Result: "string", Runtime: "auto"}
	if mode, _ := chooseMode(str, nil, ""); mode != "linked" {
		t.Fatalf("string binding mode = %q, want linked", mode)
	}
}

func TestStripLeanComments(t *testing.T) {
	depth := 0
	cases := map[string]string{
		"-- sorry in a line comment":          "",
		"theorem x : True := by sorry":        "theorem x : True := by sorry",
		"/- block sorry -/ theorem x : True":  " theorem x : True",
		"theorem y : True := by trivial -- x": "theorem y : True := by trivial ",
	}
	for in, want := range cases {
		depth = 0
		if got := stripLeanComments(in, &depth); got != want {
			t.Errorf("strip(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScaffoldWritesCompilingFiles(t *testing.T) {
	// The scaffold is exercised end to end by scripts/e2e.sh; here we only
	// guard against an empty or unparseable template.
	for name, tpl := range map[string]string{
		"policy":   policyTemplate,
		"proofs":   proofsTemplate,
		"boundary": boundaryTemplate,
	} {
		data := struct{ Module, Export, GoName, Pkg string }{"Demo", "leanport_demo_decide", "Decide", "demo"}
		out, err := render(tpl, data)
		if err != nil {
			t.Fatalf("%s template: %v", name, err)
		}
		if len(out) < 50 {
			t.Fatalf("%s template rendered suspiciously short: %q", name, out)
		}
	}
}
