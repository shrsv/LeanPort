package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }
func usageErr(format string, a ...any) error {
	return usageError{fmt.Sprintf(format, a...)}
}
func IsUsage(err error) bool {
	var u usageError
	return errors.As(err, &u)
}

func Run(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	// Global -C/--dir applies to every command.
	for len(args) > 0 {
		if args[0] == "-C" || args[0] == "--dir" {
			if len(args) < 2 {
				return usageErr("%s requires a path", args[0])
			}
			root, err = filepath.Abs(args[1])
			if err != nil {
				return err
			}
			args = args[2:]
			continue
		}
		if args[0] == "--version" {
			printVersion(os.Stdout)
			return nil
		}
		break
	}

	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	case "version":
		printVersion(os.Stdout)
		return nil
	case "init":
		return cmdInit(root, rest)
	case "new":
		return cmdNew(root, rest)
	case "check":
		return cmdCheck(root, rest)
	case "build":
		return cmdBuild(root, rest)
	case "docs":
		return cmdDocs(root, rest)
	case "doctor":
		return cmdDoctor(root, rest)
	default:
		return usageErr("unknown command %q (try: leanport help)", cmd)
	}
}

func newFlags(name string, out io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(out)
	return fs
}

// parseFlags treats --help as a successful command, not a usage error.
func parseFlags(fs *flag.FlagSet, args []string) (bool, error) {
	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		return true, nil
	}
	if err != nil {
		return false, usageErr("%v", err)
	}
	return false, nil
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `LeanPort - bring Lean 4 proven modules into your Go projects

Usage:
  leanport <command> [flags]

Commands:
  init      Set up LeanPort in an existing Go project (pins toolchains)
  new       Scaffold a new verified module (spec, proofs, boundary, binding)
  check     Type-check proofs and reject unproved holes
  build     Compile Lean to C, archive it, and generate the Go binding
  docs      Generate per-module specification files
  doctor    Diagnose missing toolchains and version mismatches
  version   Print version information

Global flags:
  -C, --dir <path>   Run as if LeanPort was started in <path>
  -h, --help         Show help
      --version      Print version

Run 'leanport <command> --help' for command flags.
`)
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "leanport %s (commit %s, built %s)\n", Version, Commit, Date)
}
