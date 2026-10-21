package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdInit(root string, args []string) error {
	fs := newFlags("init", os.Stderr)
	envKind := fs.String("env", "nix", "environment: nix or local")
	lean := fs.String("lean", defaultLeanToolchain, "Lean toolchain (v4.29.1, stable, or full name)")
	targets := fs.String("targets", "linux/amd64", "comma-separated goos/goarch targets")
	ci := fs.String("ci", "github", "CI to generate: github or none")
	name := fs.String("name", "", "project name (defaults to directory name)")
	goModule := fs.String("go-module", "", "Go module path (defaults to go.mod)")
	force := fs.Bool("force", false, "overwrite existing files")
	if done, err := parseFlags(fs, args); err != nil || done {
		return err
	}
	if *envKind != "nix" && *envKind != "local" {
		return usageErr("--env must be nix or local")
	}
	if *ci != "github" && *ci != "none" {
		return usageErr("--ci must be github or none")
	}

	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	existing, _ := loadConfig(root)
	if existing != nil && !*force {
		return fmt.Errorf("leanport.toml already exists (use --force to refresh the scaffold)")
	}

	if *name == "" {
		*name = filepath.Base(root)
	}
	if *goModule == "" {
		*goModule = detectGoModule(root)
	}
	tc := normalizeToolchain(*lean)

	cfg := &Config{
		Schema: 1,
		Project: Project{
			Name:       *name,
			LeanDir:    "lean",
			GoModule:   *goModule,
			PackageDir: "internal/verified",
			Targets:    splitTargets(*targets),
		},
		Lean: LeanCfg{Toolchain: tc},
	}
	cfg.applyDefaults()
	if existing != nil {
		// --force refreshes generated scaffolding, not the modules the user
		// has added. Preserve values the user did not explicitly override.
		cfg.Bindings = existing.Bindings
		if !visited["lean"] {
			cfg.Lean.Toolchain = existing.Lean.Toolchain
			tc = existing.Lean.Toolchain
		}
		if !visited["name"] {
			cfg.Project.Name = existing.Project.Name
		}
		if !visited["go-module"] {
			cfg.Project.GoModule = existing.Project.GoModule
		}
		if !visited["targets"] {
			cfg.Project.Targets = existing.Project.Targets
		}
	}

	lakeRel := filepath.Join("lean", "lakefile.toml")
	files := map[string]string{
		filepath.Join("lean", "lean-toolchain"): "leanprover/lean4:" + strings.TrimPrefix(tc, "leanprover/lean4:") + "\n",
		lakeRel:                                 fmt.Sprintf("name = %q\n\n# One lean_lib is appended per module by `leanport new`.\n", sanitizeLakeName(cfg.Project.Name)),
		"AGENTS.md":                             mustRender(agentsTemplate, cfg),
	}
	if *envKind == "nix" {
		files["flake.nix"] = flakeTemplate
		files[".envrc"] = "use flake\n"
	}
	if *ci == "github" {
		files[filepath.Join(".github", "workflows", "verify.yml")] = ciTemplate
	}

	lakePath := filepath.Join(root, lakeRel)
	lakeData, _ := os.ReadFile(lakePath)
	keepLake := existing != nil && strings.Contains(string(lakeData), "[[lean_lib]]")

	for rel, content := range files {
		if rel == lakeRel && keepLake {
			continue
		}
		if err := writeFile(root, rel, content, *force); err != nil {
			return err
		}
	}
	if err := ensureGitignore(root); err != nil {
		return err
	}
	if err := saveConfig(root, cfg); err != nil {
		return err
	}

	fmt.Printf("LeanPort initialised in %s\n", root)
	fmt.Printf("  Lean toolchain : %s\n", tc)
	fmt.Printf("  Go module      : %s\n", *goModule)
	fmt.Printf("  targets        : %s\n", strings.Join(cfg.Project.Targets, ", "))
	fmt.Printf("  environment    : %s\n\n", *envKind)
	fmt.Println("Next:")
	fmt.Println("  leanport new <module>   # scaffold a verified module")
	fmt.Println("  leanport check          # prove it")
	fmt.Println("  leanport build          # generate the Go binding")
	return nil
}

func detectGoModule(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func normalizeToolchain(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		v = defaultLeanToolchain
	}
	if strings.HasPrefix(v, "leanprover/lean4:") {
		return strings.TrimPrefix(v, "leanprover/lean4:")
	}
	return strings.TrimPrefix(v, "v")
}

func splitTargets(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func sanitizeLakeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "leanport_project"
	}
	return b.String()
}

func writeFile(root, rel, content string, force bool) error {
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", rel)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func mustRender(tpl string, data any) string {
	s, err := render(tpl, data)
	if err != nil {
		panic(err)
	}
	return s
}

// ensureGitignore appends the generated/intermediate directories without
// clobbering an existing .gitignore. Generated Go packages are intentionally
// not ignored: committing them keeps consumers from needing LeanPort.
func ensureGitignore(root string) error {
	const marker = "# LeanPort generated / intermediate files"
	block := marker + "\n" +
		".leanport/\n" +
		".lake/\n" +
		".elan/\n" +
		".direnv/\n"
	path := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return os.WriteFile(path, []byte(block), 0o644)
	}
	if err != nil {
		return err
	}
	if strings.Contains(string(data), marker) {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString("\n" + block)
	return err
}
