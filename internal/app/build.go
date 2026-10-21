package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var deadPathSymbols = map[string]bool{
	"initialize_Init":                   true,
	"lean_inc_heartbeat":                true,
	"mi_malloc_small":                   true,
	"lean_dec_ref_cold":                 true,
	"lean_internal_panic_out_of_memory": true,
	"lean_notify_assert":                true,
}

func cmdBuild(root string, args []string) error {
	fs := newFlags("build", os.Stderr)
	target := fs.String("target", "", "build only this goos/goarch target")
	verify := fs.Bool("verify", true, "run go build ./... after generating")
	clean := fs.Bool("clean", false, "remove previous build intermediates")
	forced := fs.String("runtime", "", "force runtime mode: none or linked")
	if done, err := parseFlags(fs, args); err != nil || done {
		return err
	}
	e, err := loadEnv(root)
	if err != nil {
		return err
	}
	if *clean {
		_ = os.RemoveAll(filepath.Join(root, ".leanport"))
	}
	if err := checkProject(e); err != nil {
		return err
	}

	goos, goarch := runtime.GOOS, runtime.GOARCH
	if *target != "" {
		parts := strings.Split(*target, "/")
		if len(parts) != 2 {
			return usageErr("--target must be goos/goarch")
		}
		goos, goarch = parts[0], parts[1]
		if goos != runtime.GOOS || goarch != runtime.GOARCH {
			return fmt.Errorf("target %s/%s differs from the host %s/%s; cross-building Lean artifacts needs a target C toolchain (not in v0.1)", goos, goarch, runtime.GOOS, runtime.GOARCH)
		}
	}

	buildDir := filepath.Join(root, ".leanport", "build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return err
	}
	objects, undefined, err := compileProject(e, buildDir)
	if err != nil {
		return err
	}
	initSyms, err := moduleInitializers(objects)
	if err != nil {
		return err
	}

	byPkg := map[string][]Binding{}
	var pkgs []string
	for _, b := range e.cfg.Bindings {
		if _, ok := byPkg[b.Package]; !ok {
			pkgs = append(pkgs, b.Package)
		}
		byPkg[b.Package] = append(byPkg[b.Package], b)
	}
	sort.Strings(pkgs)

	for _, pkg := range pkgs {
		bindings := byPkg[pkg]
		pkgMode := "none"
		for _, b := range bindings {
			m, err := chooseMode(b, undefined, *forced)
			if err != nil {
				return err
			}
			if m == "linked" {
				pkgMode = "linked"
			}
		}
		if pkgMode == "linked" {
			for _, b := range bindings {
				t, err := parseType(b.Result)
				if err != nil {
					return err
				}
				if t.Kind != KindBool && t.Kind != KindScalar && t.Kind != KindEnum && t.Kind != KindString && t.Kind != KindBytes {
					return fmt.Errorf("binding %s.%s: result type %s is not supported by the generator yet", b.Package, b.GoName, typeName(t))
				}
				for _, a := range b.Args {
					at, err := parseType(a)
					if err != nil {
						return err
					}
					if at.Kind != KindBool && at.Kind != KindScalar && at.Kind != KindEnum && at.Kind != KindString && at.Kind != KindBytes {
						return fmt.Errorf("binding %s.%s: argument type %s is not supported by the generator yet", b.Package, b.GoName, typeName(at))
					}
				}
			}
		}

		var glueObjs []string
		for _, b := range bindings {
			glue, err := glueFor(b, pkgMode, initSyms)
			if err != nil {
				return err
			}
			gluePath := filepath.Join(buildDir, fmt.Sprintf("glue_%s_%s_%s.c", pkg, b.Module, b.GoName))
			if err := os.WriteFile(gluePath, []byte(glue), 0o644); err != nil {
				return err
			}
			glueObj := strings.TrimSuffix(gluePath, ".c") + ".o"
			cflags, err := e.cflags()
			if err != nil {
				return err
			}
			ccArgs := []string{"-c", "-O2", "-fPIC", "-ffunction-sections", "-fdata-sections"}
			if pkgMode == "linked" {
				ccArgs = append(ccArgs, cflags...)
			}
			ccArgs = append(ccArgs, gluePath, "-o", glueObj)
			if err := e.run(root, e.cc(), ccArgs...); err != nil {
				return fmt.Errorf("compiling generated glue: %w", err)
			}
			glueObjs = append(glueObjs, glueObj)
		}

		archiveRel := filepath.Join(e.cfg.Project.PackageDir, pkg, "native", goos+"_"+goarch, "lib"+bindings[0].Module+".a")
		archivePath := filepath.Join(root, archiveRel)
		if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
			return err
		}
		if pkgMode == "linked" {
			if goos != "linux" {
				return fmt.Errorf("runtime-linked mode currently supports linux only (target %s)", goos)
			}
			if err := linkRuntime(e, buildDir, glueObjs, objects, archivePath); err != nil {
				return err
			}
		} else {
			if err := runAr(archivePath, append(append([]string{}, objects...), glueObjs...)); err != nil {
				return err
			}
		}
		fmt.Printf("→ %s (%s mode, %s)\n", archiveRel, pkgMode, humanSize(archivePath))
		if err := generatePackage(e, bindings, []string{archiveRel}, goos, goarch, pkgMode); err != nil {
			return err
		}
	}

	if *verify {
		fmt.Println("→ go build ./...")
		if err := e.run(root, "go", "build", "./..."); err != nil {
			return fmt.Errorf("generated binding does not link: %w", err)
		}
	}
	fmt.Println("✓ build complete")
	return nil
}

func compileProject(e *env, buildDir string) ([]string, []string, error) {
	cflags, err := e.cflags()
	if err != nil {
		return nil, nil, err
	}
	files, err := e.generatedC()
	if err != nil {
		return nil, nil, err
	}
	var objects []string
	for i, f := range files {
		o := filepath.Join(buildDir, fmt.Sprintf("mod_%03d_%s.o", i, sanitizeFile(filepath.Base(f))))
		args := []string{"-c", "-O2", "-fPIC", "-ffunction-sections", "-fdata-sections"}
		args = append(args, cflags...)
		args = append(args, f, "-o", o)
		if err := e.run(e.leanDir(), e.cc(), args...); err != nil {
			return nil, nil, fmt.Errorf("compiling %s: %w", f, err)
		}
		objects = append(objects, o)
	}
	undef, err := undefinedSymbols(objects)
	if err != nil {
		return nil, nil, err
	}
	return objects, undef, nil
}

func undefinedSymbols(objects []string) ([]string, error) {
	// A symbol may be undefined in one object and defined in another; resolve
	// across the whole set before deciding whether the Lean runtime is needed.
	defined := map[string]bool{}
	for _, o := range objects {
		out, err := exec.Command("nm", "-g", "--defined-only", o).Output()
		if err != nil {
			return nil, fmt.Errorf("nm %s: %w", o, err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				defined[fields[2]] = true
			}
		}
	}
	seen := map[string]bool{}
	for _, o := range objects {
		out, err := exec.Command("nm", "-u", o).Output()
		if err != nil {
			return nil, fmt.Errorf("nm -u %s: %w", o, err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			sym := fields[len(fields)-1]
			if !defined[sym] {
				seen[sym] = true
			}
		}
	}
	var out []string
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}

func moduleInitializers(objects []string) ([]string, error) {
	seen := map[string]bool{}
	for _, o := range objects {
		out, err := exec.Command("nm", "-g", "--defined-only", o).Output()
		if err != nil {
			return nil, fmt.Errorf("nm %s: %w", o, err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 || fields[1] != "T" {
				continue
			}
			sym := fields[2]
			if strings.HasPrefix(sym, "initialize_") && sym != "initialize_Init" && !strings.HasPrefix(sym, "initialize_libuv") {
				seen[sym] = true
			}
		}
	}
	var out []string
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}

func isLeanRuntimeSymbol(s string) bool {
	return strings.HasPrefix(s, "lean_") || strings.HasPrefix(s, "mi_") || strings.HasPrefix(s, "initialize_")
}

func chooseMode(b Binding, undefined []string, forced string) (string, error) {
	mode := forced
	if mode == "" {
		mode = b.Runtime
	}
	if mode == "" || mode == "auto" {
		for _, a := range b.Args {
			t, err := parseType(a)
			if err != nil {
				return "", err
			}
			if t.NeedsRuntime() {
				return "linked", nil
			}
		}
		if b.Result != "" {
			t, err := parseType(b.Result)
			if err != nil {
				return "", err
			}
			if t.NeedsRuntime() {
				return "linked", nil
			}
		}
		for _, s := range undefined {
			if isLeanRuntimeSymbol(s) && !deadPathSymbols[s] {
				return "linked", nil
			}
		}
		return "none", nil
	}
	if mode != "none" && mode != "linked" {
		return "", usageErr("runtime must be auto, none or linked (got %q)", mode)
	}
	return mode, nil
}

func linkRuntime(e *env, buildDir string, glueObjs []string, objects []string, archivePath string) error {
	prefix, err := e.leanPrefix()
	if err != nil {
		return err
	}
	merged := filepath.Join(buildDir, "merged_"+sanitizeFile(filepath.Base(archivePath))+".o")
	args := []string{"-r", "-o", merged}
	args = append(args, glueObjs...)
	args = append(args, objects...)
	args = append(args,
		"-L"+filepath.Join(prefix, "lib", "lean"),
		"-L"+filepath.Join(prefix, "lib"),
		"--start-group", "-lleancpp", "-lLean", "-lStd", "-lInit", "-lleanrt",
		"-lgmp", "-luv", "-lc++", "-lc++abi", "--end-group",
	)
	if err := e.run(buildDir, "ld", args...); err != nil {
		return fmt.Errorf("linking Lean runtime into archive: %w", err)
	}
	return runAr(archivePath, []string{merged})
}

func runAr(path string, objects []string) error {
	_ = os.Remove(path)
	args := append([]string{"rcs", path}, objects...)
	cmd := exec.Command("ar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ar: %w", err)
	}
	return nil
}

func humanSize(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	n := fi.Size()
	switch {
	case n > 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n > 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func sanitizeFile(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

type manifest struct {
	Schema    int            `json:"schema"`
	Generator string         `json:"generator"`
	Lean      manifestLean   `json:"lean"`
	Target    manifestTarget `json:"target"`
	Mode      string         `json:"mode"`
	Exports   []manifestExp  `json:"exports"`
	Artifacts []manifestArt  `json:"artifacts"`
}

type manifestLean struct {
	Toolchain string `json:"toolchain"`
	Compiler  string `json:"compiler,omitempty"`
}
type manifestTarget struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}
type manifestExp struct {
	CSymbol string   `json:"c_symbol"`
	Go      string   `json:"go"`
	Args    []string `json:"args"`
	Result  string   `json:"result"`
}
type manifestArt struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func writeManifest(e *env, pkg string, bindings []Binding, archives []string, goos, goarch, mode string) (string, error) {
	m := manifest{
		Schema:    1,
		Generator: "leanport " + Version,
		Lean:      manifestLean{Toolchain: e.cfg.Lean.Toolchain},
		Target:    manifestTarget{GOOS: goos, GOARCH: goarch},
		Mode:      mode,
	}
	for _, b := range bindings {
		m.Exports = append(m.Exports, manifestExp{CSymbol: b.Export, Go: b.GoName, Args: b.Args, Result: b.Result})
	}
	for _, a := range archives {
		data, err := os.ReadFile(filepath.Join(e.root, a))
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		m.Artifacts = append(m.Artifacts, manifestArt{Path: a, SHA256: hex.EncodeToString(sum[:])})
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(e.root, e.cfg.Project.PackageDir, pkg, "MANIFEST.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	var sums []string
	for _, a := range m.Artifacts {
		sums = append(sums, a.SHA256)
	}
	return strings.Join(sums, ","), nil
}
