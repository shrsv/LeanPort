package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Version metadata, injected from main via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type Config struct {
	Schema   int       `toml:"schema"`
	Project  Project   `toml:"project"`
	Lean     LeanCfg   `toml:"lean"`
	Bindings []Binding `toml:"binding"`
}

type Project struct {
	Name       string   `toml:"name"`
	LeanDir    string   `toml:"lean_dir"`
	GoModule   string   `toml:"go_module"`
	PackageDir string   `toml:"package_dir"`
	Targets    []string `toml:"targets"`
}

type LeanCfg struct {
	Toolchain string `toml:"toolchain"`
}

type Binding struct {
	Package string           `toml:"package"`
	Module  string           `toml:"module"`
	Export  string           `toml:"export"`
	GoName  string           `toml:"go_name"`
	Args    []string         `toml:"args"`
	Result  string           `toml:"result"`
	Runtime string           `toml:"runtime"`
	Actions map[string]uint8 `toml:"actions"`
}

func (c *Config) applyDefaults() {
	if c.Schema == 0 {
		c.Schema = 1
	}
	if c.Project.LeanDir == "" {
		c.Project.LeanDir = "lean"
	}
	if c.Project.PackageDir == "" {
		c.Project.PackageDir = "internal/verified"
	}
	if len(c.Project.Targets) == 0 {
		c.Project.Targets = []string{"linux/amd64"}
	}
	for i := range c.Bindings {
		if c.Bindings[i].Runtime == "" {
			c.Bindings[i].Runtime = "auto"
		}
	}
}

func (c *Config) validate() error {
	if c.Project.LeanDir == "" {
		return usageErr("project.lean_dir is empty")
	}
	if c.Project.PackageDir == "" {
		return usageErr("project.package_dir is empty")
	}
	for _, b := range c.Bindings {
		if b.Package == "" || b.Module == "" || b.Export == "" || b.GoName == "" {
			return usageErr("binding is missing package/module/export/go_name")
		}
		if b.Result == "" {
			return usageErr("binding %s: result is required in v0.1", b.Package)
		}
		res, err := parseType(b.Result)
		if err != nil {
			return fmt.Errorf("binding %s result: %w", b.Package, err)
		}
		hasEnum := res.Kind == KindEnum
		for _, a := range b.Args {
			t, err := parseType(a)
			if err != nil {
				return fmt.Errorf("binding %s arg %q: %w", b.Package, a, err)
			}
			if t.Kind == KindEnum {
				hasEnum = true
			}
		}
		if hasEnum && len(b.Actions) == 0 {
			return usageErr("binding %s uses an action/enum type but has no actions table", b.Package)
		}
	}
	return nil
}

func configPath(root string) string { return filepath.Join(root, "leanport.toml") }

func loadConfig(root string) (*Config, error) {
	p := configPath(root)
	var c Config
	if _, err := toml.DecodeFile(p, &c); err != nil {
		if os.IsNotExist(err) {
			return nil, usageErr("no leanport.toml in %s (run: leanport init)", root)
		}
		return nil, fmt.Errorf("reading %s: %w", p, err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveConfig(root string, c *Config) error {
	f, err := os.Create(configPath(root))
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintln(f, "# LeanPort project contract. See docs/cli-reference.md.")
	fmt.Fprintln(f, "# Files under internal/verified/ are generated; edit lean/ instead.")
	fmt.Fprintln(f)
	return toml.NewEncoder(f).Encode(c)
}

// parseType turns a config type token into a Type.
func parseType(s string) (Type, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "bool":
		return Type{Kind: KindBool}, nil
	case "uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64",
		"usize", "float", "float32", "rune":
		return Type{Kind: KindScalar, Name: s}, nil
	case "string":
		return Type{Kind: KindString}, nil
	case "bytes":
		return Type{Kind: KindBytes}, nil
	}
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		if (parts[0] != "action" && parts[0] != "enum") || parts[1] == "" {
			return Type{}, fmt.Errorf("unknown tagged type %q", s)
		}
		return Type{Kind: KindEnum, Enum: parts[1]}, nil
	}
	return Type{}, fmt.Errorf("unsupported type %q (see docs/cli-reference.md)", s)
}

type Kind int

const (
	KindBool Kind = iota
	KindScalar
	KindEnum
	KindString
	KindBytes
)

type Type struct {
	Kind Kind
	Name string // scalar name
	Enum string // enum type name
}

func (t Type) NeedsRuntime() bool { return t.Kind == KindString || t.Kind == KindBytes }

// CType is the C type of the argument as it crosses the Lean export ABI.
func (t Type) CType() string {
	switch t.Kind {
	case KindBool:
		return "uint8_t"
	case KindEnum:
		return "uint8_t"
	case KindScalar:
		switch t.Name {
		case "uint8":
			return "uint8_t"
		case "uint16":
			return "uint16_t"
		case "uint32":
			return "uint32_t"
		case "uint64":
			return "uint64_t"
		case "int8":
			return "int8_t"
		case "int16":
			return "int16_t"
		case "int32":
			return "int32_t"
		case "int64":
			return "int64_t"
		case "usize":
			return "size_t"
		case "float":
			return "double"
		case "float32":
			return "float"
		case "rune":
			return "uint32_t"
		}
	case KindString, KindBytes:
		return "lean_object*"
	}
	return "void"
}

// GoType is the public Go type.
func (t Type) GoType(c *Config, b Binding) string {
	switch t.Kind {
	case KindBool:
		return "bool"
	case KindEnum:
		return t.Enum
	case KindScalar:
		switch t.Name {
		case "usize":
			return "uint"
		case "float":
			return "float64"
		default:
			return t.Name
		}
	case KindString:
		return "string"
	case KindBytes:
		return "[]byte"
	}
	return "any"
}
