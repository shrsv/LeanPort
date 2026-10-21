package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var proofHole = regexp.MustCompile(`\b(sorry|admit)\b`)

func cmdCheck(root string, args []string) error {
	fs := newFlags("check", os.Stderr)
	if done, err := parseFlags(fs, args); err != nil || done {
		return err
	}
	e, err := loadEnv(root)
	if err != nil {
		return err
	}
	if err := checkProject(e); err != nil {
		return err
	}
	fmt.Printf("✓ checked %d module(s); %d export(s) verified\n", distinctModules(e.cfg), len(e.cfg.Bindings))
	return nil
}

func distinctModules(c *Config) int {
	seen := map[string]bool{}
	for _, b := range c.Bindings {
		seen[b.Module] = true
	}
	return len(seen)
}

func checkProject(e *env) error {
	leanDir := e.leanDir()
	if _, err := os.Stat(leanDir); err != nil {
		return fmt.Errorf("lean directory %q not found (run leanport init)", leanDir)
	}

	if err := scanProofHoles(leanDir); err != nil {
		return err
	}

	targets := e.buildTargets()
	args := []string{"build"}
	args = append(args, targets...)
	fmt.Printf("→ lake %s\n", strings.Join(args, " "))
	if err := e.run(leanDir, "lake", args...); err != nil {
		return fmt.Errorf("Lean build failed: %w", err)
	}

	files, err := e.generatedC()
	if err != nil {
		return err
	}
	contents := make([]string, len(files))
	for i, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		contents[i] = string(data)
	}
	for _, b := range e.cfg.Bindings {
		found := false
		symbol := regexp.MustCompile(`\b` + regexp.QuoteMeta(b.Export) + `\s*\(`)
		for _, data := range contents {
			if exportDecl.MatchString(data) && symbol.MatchString(data) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("export %q for package %q not found in generated C; check the @[export %s] declaration", b.Export, b.Package, b.Export)
		}
	}
	return nil
}

var exportDecl = regexp.MustCompile(`LEAN_EXPORT`)

func scanProofHoles(leanDir string) error {
	var bad []string
	err := filepath.WalkDir(leanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".lake" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".lean") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		depth := 0
		for i, line := range strings.Split(string(data), "\n") {
			if code := stripLeanComments(line, &depth); proofHole.MatchString(code) {
				bad = append(bad, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(bad) > 0 {
		return fmt.Errorf("unproved holes found (remove sorry/admit):\n  %s", strings.Join(bad, "\n  "))
	}
	return nil
}

// stripLeanComments removes line and block comments so documentation that
// mentions an unfinished-proof tactic does not fail the check.
func stripLeanComments(line string, depth *int) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		if *depth > 0 {
			switch {
			case strings.HasPrefix(line[i:], "/-"):
				*depth++
				i += 2
			case strings.HasPrefix(line[i:], "-/"):
				*depth--
				i += 2
			default:
				i++
			}
			continue
		}
		switch {
		case strings.HasPrefix(line[i:], "--"):
			return out.String()
		case strings.HasPrefix(line[i:], "/-"):
			*depth++
			i += 2
		default:
			out.WriteByte(line[i])
			i++
		}
	}
	return out.String()
}
