package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const defaultLeanToolchain = "v4.29.1"

type env struct {
	root   string
	cfg    *Config
	prefix string
}

func loadEnv(root string) (*env, error) {
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}
	return &env{root: root, cfg: cfg}, nil
}

func (e *env) leanDir() string { return filepath.Join(e.root, e.cfg.Project.LeanDir) }

func (e *env) cc() string {
	if v := os.Getenv("LEANPORT_CC"); v != "" {
		return v
	}
	return "cc"
}

func (e *env) run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func (e *env) output(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *env) leanPrefix() (string, error) {
	if e.prefix != "" {
		return e.prefix, nil
	}
	out, err := e.output(e.leanDir(), "lean", "--print-prefix")
	if err != nil {
		return "", fmt.Errorf("locating Lean (is elan installed and lean-toolchain valid?): %w", err)
	}
	if out == "" {
		return "", fmt.Errorf("lean --print-prefix returned nothing")
	}
	e.prefix = out
	return out, nil
}

func (e *env) leanc() (string, error) {
	p, err := e.leanPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(p, "bin", "leanc"), nil
}

func (e *env) cflags() ([]string, error) {
	leanc, err := e.leanc()
	if err != nil {
		return nil, err
	}
	out, err := e.output(e.leanDir(), leanc, "--print-cflags")
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
}

// generatedC returns every Lean-generated C file for the project, excluding
// toolchain and Lake build metadata.
func (e *env) generatedC() ([]string, error) {
	ir := filepath.Join(e.leanDir(), ".lake", "build", "ir")
	var files []string
	err := filepath.WalkDir(ir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".c") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("no generated C under %s (run leanport check first): %w", ir, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no generated C under %s", ir)
	}
	return files, nil
}

// buildTargets returns the Lake targets for all declared modules.
func (e *env) buildTargets() []string {
	var targets []string
	seen := map[string]bool{}
	for _, b := range e.cfg.Bindings {
		for _, suffix := range []string{".Policy", ".Proofs", ".Boundary"} {
			t := "LeanPort." + b.Module + suffix
			if !seen[t] {
				seen[t] = true
				targets = append(targets, t)
			}
		}
	}
	return targets
}
