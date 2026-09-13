package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/policy"
)

func runResolve(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	type input struct{ path, constraints string }
	var inputs []input
	var output string
	fs.Func("layer", "literal configuration file, least specific first (repeatable)", func(path string) error {
		if path == "" {
			return errors.New("--layer requires a nonempty path")
		}
		inputs = append(inputs, input{path: path})
		return nil
	})
	fs.Func("constraints", "constraint file for the preceding --layer", func(path string) error {
		if len(inputs) == 0 {
			return errors.New("--constraints must follow --layer")
		}
		if path == "" {
			return errors.New("--constraints requires a nonempty path")
		}
		if inputs[len(inputs)-1].constraints != "" {
			return errors.New("only one --constraints file is allowed per layer")
		}
		inputs[len(inputs)-1].constraints = path
		return nil
	})
	fs.StringVar(&output, "output", "", "write resolved YAML to this file instead of stdout")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: ghrepocfg resolve --layer FILE [--constraints FILE] [--layer FILE ...] [--output FILE]\n\nMaterialize ordered literal configurations locally. No GitHub credentials are needed.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return reportError(stderr, err)
	}
	if fs.NArg() != 0 {
		return reportError(stderr, errors.New("resolve accepts layers through repeated --layer FILE options"))
	}
	layers := make([]policy.Layer, 0, len(inputs))
	for i, input := range inputs {
		b, err := os.ReadFile(input.path)
		if err != nil {
			return reportError(stderr, err)
		}
		layer := policy.Layer{Name: fmt.Sprintf("%d (%s)", i+1, input.path), Config: b}
		if input.constraints != "" {
			layer.Constraints, err = os.ReadFile(input.constraints)
			if err != nil {
				return reportError(stderr, err)
			}
			if layer.Constraints == nil {
				layer.Constraints = []byte{}
			}
		}
		layers = append(layers, layer)
	}
	b, err := policy.Resolve(layers)
	if err != nil {
		return reportError(stderr, err)
	}
	if output != "" {
		err = atomicWrite(output, b)
	} else {
		_, err = stdout.Write(b)
	}
	if err != nil {
		return reportError(stderr, err)
	}
	return exitOK
}

func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var structured bool
	fs.BoolVar(&structured, "json", false, "structured effective configuration changes")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: ghrepocfg diff [--json] OLD.yaml NEW.yaml\n\nCompare literal configurations locally. Exit codes: 0 equal, 1 error, 2 changed.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return reportError(stderr, err)
	}
	if fs.NArg() != 2 {
		return reportError(stderr, errors.New("diff requires two configuration files: OLD.yaml NEW.yaml"))
	}
	before, err := config.Load(fs.Arg(0))
	if err != nil {
		return reportError(stderr, fmt.Errorf("load %s: %w", fs.Arg(0), err))
	}
	after, err := config.Load(fs.Arg(1))
	if err != nil {
		return reportError(stderr, fmt.Errorf("load %s: %w", fs.Arg(1), err))
	}
	result, err := policy.Compare(before, after)
	if err != nil {
		return reportError(stderr, err)
	}
	if structured {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(result)
	} else {
		err = printEffectiveDiff(stdout, result)
	}
	if err != nil {
		return reportError(stderr, err)
	}
	if result.Changed {
		return exitDrift
	}
	return exitOK
}
func printEffectiveDiff(w io.Writer, result policy.Diff) error {
	s := styleFor(w)
	if !result.Changed {
		_, err := fmt.Fprintln(w, s.green(s.bold("No effective configuration changes.")))
		return err
	}
	if _, err := fmt.Fprintln(w, s.bold("Effective configuration changes:")); err != nil {
		return err
	}
	for _, c := range result.Changes {
		if _, err := fmt.Fprintf(w, "\n  %s\n    ", s.cyan(c.Path)); err != nil {
			return err
		}
		var err error
		switch c.Operation {
		case "add":
			_, err = fmt.Fprintf(w, "%s %s\n", s.green("add:"), s.green(string(c.After)))
		case "remove":
			_, err = fmt.Fprintf(w, "%s %s\n", s.red("remove:"), s.red(string(c.Before)))
		default:
			_, err = fmt.Fprintf(w, "%s %s %s\n", s.yellow(string(c.Before)), s.dim("->"), s.green(string(c.After)))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
