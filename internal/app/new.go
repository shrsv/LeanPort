package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdNew(root string, args []string) error {
	fs := newFlags("new", os.Stderr)
	pkg := fs.String("package", "", "generated Go package name (default: lower-case module)")
	export := fs.String("export", "", "C symbol for @[export] (default: leanport_<module>_decide)")
	goName := fs.String("go-name", "Decide", "Go function name")
	if done, err := parseFlags(fs, args); err != nil || done {
		return err
	}
	if fs.NArg() != 1 {
		return usageErr("usage: leanport new <module>")
	}
	module := fs.Arg(0)
	if !validIdent(module) {
		return usageErr("module must be a Lean identifier (letters, digits, underscore), got %q", module)
	}
	if *pkg == "" {
		*pkg = strings.ToLower(module)
	}
	if !validIdent(*pkg) {
		return usageErr("package must be a Go identifier, got %q", *pkg)
	}
	if *export == "" {
		*export = "leanport_" + strings.ToLower(module) + "_" + strings.ToLower(*goName)
	}

	e, err := loadEnv(root)
	if err != nil {
		return err
	}
	for _, b := range e.cfg.Bindings {
		if b.Module == module {
			return fmt.Errorf("module %s already exists in leanport.toml", module)
		}
	}

	data := struct {
		Module string
		Export string
		GoName string
		Pkg    string
	}{module, *export, *goName, *pkg}

	base := filepath.Join("lean", "LeanPort", module)
	files := map[string]string{
		filepath.Join(base, "Policy.lean"):   mustRender(policyTemplate, data),
		filepath.Join(base, "Proofs.lean"):   mustRender(proofsTemplate, data),
		filepath.Join(base, "Boundary.lean"): mustRender(boundaryTemplate, data),
		filepath.Join(base, "SPEC.md"):       mustRender(specTemplate, data),
	}
	for rel, content := range files {
		if err := writeFile(root, rel, content, false); err != nil {
			return err
		}
	}
	if err := appendLeanLib(e.leanDir(), module); err != nil {
		return err
	}

	e.cfg.Bindings = append(e.cfg.Bindings, Binding{
		Package: *pkg,
		Module:  module,
		Export:  *export,
		GoName:  *goName,
		Args:    []string{"bool", "uint8"},
		Result:  "bool",
		Runtime: "auto",
	})
	if err := saveConfig(root, e.cfg); err != nil {
		return err
	}

	fmt.Printf("created lean/LeanPort/%s/{Policy,Proofs,Boundary}.lean\n", module)
	fmt.Printf("added binding %s.%s -> %s\n", *pkg, *goName, *export)
	fmt.Printf("\nEdit the Lean files, then: leanport check && leanport build\n")
	return nil
}

func validIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func appendLeanLib(leanDir, module string) error {
	path := filepath.Join(leanDir, "lakefile.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading lakefile: %w", err)
	}
	block := fmt.Sprintf("\n[[lean_lib]]\nname = %q\nsrcDir = \".\"\nglobs = [\"LeanPort.%s.*\"]\n", "LeanPort."+module, module)
	// Avoid duplicating the library block on re-runs.
	if strings.Contains(string(data), block) || strings.Contains(string(data), fmt.Sprintf("name = %q", "LeanPort."+module)) {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(block)
	return err
}
