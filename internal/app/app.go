// Package app implements ghrepocfg's command-line behavior.
package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/exportconfig"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/reconcile"
)

const (
	exitOK    = 0
	exitError = 1
	exitDrift = 2
)

type options struct {
	repo, config                     string
	verbose, json, dryRun, yes, full bool
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		usage(stderr)
		return exitError
	}
	switch args[0] {
	case "version", "--version", "-version":
		fmt.Fprintln(stdout, version)
		return exitOK
	case "help", "--help", "-h":
		usage(stdout)
		return exitOK
	case "apply":
		return runApply(args[1:], stdin, stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	default:
		return reportError(stderr, fmt.Errorf("unknown command %q (expected apply or export)", args[0]))
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  ghrepocfg export [--repo OWNER/REPO] [--config PATH] [--full] [--dry-run]
  ghrepocfg apply  [--repo OWNER/REPO] [--config PATH] [--dry-run] [-y]

Common options:
  -R, --repo OWNER/REPO  target repository (or GHREPOCFG_REPO)
      --config PATH      YAML file (or GHREPOCFG_CONFIG)
  -v, --verbose          additional diagnostics
      --json             structured dry-run output
  -h, --help             show command help`)
}

func parseFlags(command string, args []string, stderr io.Writer) (options, error) {
	var o options
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.repo, "repo", "", "target owner/repo")
	fs.StringVar(&o.repo, "R", "", "target owner/repo")
	fs.StringVar(&o.config, "config", "", "configuration path")
	fs.BoolVar(&o.verbose, "verbose", false, "verbose diagnostics")
	fs.BoolVar(&o.verbose, "v", false, "verbose diagnostics")
	fs.BoolVar(&o.json, "json", false, "JSON output")
	fs.BoolVar(&o.dryRun, "dry-run", false, "plan without writing")
	if command == "apply" {
		fs.BoolVar(&o.yes, "yes", false, "approve all changes")
		fs.BoolVar(&o.yes, "y", false, "approve all changes")
	}
	if command == "export" {
		fs.BoolVar(&o.full, "full", false, "replace management scope")
	}
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() != 0 {
		return o, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if o.repo == "" {
		o.repo = os.Getenv("GHREPOCFG_REPO")
	}
	if o.config == "" {
		o.config = os.Getenv("GHREPOCFG_CONFIG")
	}
	if o.json && !o.dryRun {
		return o, errors.New("--json is supported with --dry-run in v1")
	}
	return o, nil
}

func runApply(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	outStyle, errStyle := styleFor(stdout), styleFor(stderr)
	o, err := parseFlags("apply", args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return reportError(stderr, err)
	}
	owner, repo, _, err := resolveRepo(o.repo)
	if err != nil {
		return reportError(stderr, err)
	}
	path := o.config
	if path == "" {
		root, ok := gitRoot()
		if ok {
			path = filepath.Join(root, ".ghrepocfg.yaml")
		} else {
			path = ".ghrepocfg.yaml"
		}
	}
	desired, err := config.Load(path)
	if err != nil {
		return reportError(stderr, fmt.Errorf("load %s: %w", path, err))
	}
	token, source, err := resolveToken()
	if err != nil {
		return reportError(stderr, err)
	}
	if o.verbose {
		fmt.Fprintf(stderr, "%s %s\n", errStyle.dim("Authenticated using"), errStyle.cyan(source))
	}
	client := github.NewClient(token)
	scope := scopeFor(desired, o.verbose)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	state, err := client.Read(ctx, owner, repo, scope)
	if err != nil {
		return reportError(stderr, actionableAPIError(err))
	}
	plan := reconcile.Build(owner, repo, desired, state, client, o.verbose)
	if o.json {
		writeJSON(stdout, plan)
		if plan.Drift {
			return exitDrift
		}
		return exitOK
	}
	printPlan(stdout, plan)
	if o.dryRun {
		if plan.Drift {
			return exitDrift
		}
		return exitOK
	}
	if !plan.Drift {
		return exitOK
	}
	if !o.yes && !confirm(stdin, stderr, len(plan.Changes)) {
		fmt.Fprintln(stderr, errStyle.yellow("Apply cancelled."))
		return exitError
	}
	succeeded, failed := plan.Execute(ctx)
	for _, path := range succeeded {
		fmt.Fprintf(stdout, "%s %s\n", outStyle.green("Applied"), outStyle.cyan(path))
	}
	if len(failed) > 0 {
		fmt.Fprintf(stderr, "\n%s\n", errStyle.red(errStyle.bold(fmt.Sprintf("%d change(s) failed:", len(failed)))))
		for _, f := range failed {
			fmt.Fprintf(stderr, "  %s: %s\n", errStyle.red(f.Path), f.Error)
		}
		return exitError
	}
	fmt.Fprintln(stdout, outStyle.green(outStyle.bold(fmt.Sprintf("Applied %d change(s).", len(succeeded)))))
	return exitOK
}

func runExport(args []string, stdout, stderr io.Writer) int {
	errStyle := styleFor(stderr)
	o, err := parseFlags("export", args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return reportError(stderr, err)
	}
	owner, repo, inferredRoot, err := resolveRepo(o.repo)
	if err != nil {
		return reportError(stderr, err)
	}
	destination := o.config
	if destination == "" && inferredRoot != "" {
		destination = filepath.Join(inferredRoot, ".ghrepocfg.yaml")
	}
	if o.dryRun && destination == "" {
		return reportError(stderr, errors.New("export --dry-run requires a file destination; use --config PATH"))
	}
	var base *config.Config
	if destination != "" && !o.full {
		if _, err := os.Stat(destination); err == nil {
			base, err = config.Load(destination)
			if err != nil {
				return reportError(stderr, fmt.Errorf("load %s: %w", destination, err))
			}
		} else if !os.IsNotExist(err) {
			return reportError(stderr, err)
		}
	}
	full := o.full || base == nil
	token, source, err := resolveToken()
	if err != nil {
		return reportError(stderr, err)
	}
	if o.verbose {
		fmt.Fprintf(stderr, "%s %s\n", errStyle.dim("Authenticated using"), errStyle.cyan(source))
	}
	scope := github.ReadScope{Verbose: o.verbose}
	if full {
		scope.Repository = true
		scope.Security = true
		scope.Actions = true
		scope.Collaborators = true
		scope.Teams = true
		scope.Rulesets = true
	} else {
		scope = scopeFor(base, o.verbose)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := github.NewClient(token)
	state, err := client.Read(ctx, owner, repo, scope)
	if err != nil {
		return reportError(stderr, actionableAPIError(err))
	}
	var exported *config.Config
	if full {
		exported = exportconfig.FromState(state)
	} else {
		exported = exportconfig.ScopedFromState(base, state)
	}
	b, err := config.Marshal(exported)
	if err != nil {
		return reportError(stderr, err)
	}
	for _, w := range state.Warnings {
		fmt.Fprintf(stderr, "%s %s\n", errStyle.yellow(errStyle.bold("warning:")), w)
	}
	if o.verbose {
		for _, f := range state.UnknownFields {
			fmt.Fprintf(stderr, "%s %s\n", errStyle.dim("unmanaged GitHub repository response field:"), errStyle.cyan(f))
		}
	}
	if o.dryRun {
		changes := diffConfig(base, exported)
		result := map[string]any{"repository": owner + "/" + repo, "changed": len(changes) > 0, "changes": changes, "warnings": state.Warnings}
		if o.json {
			writeJSON(stdout, result)
		} else {
			printExportDiff(stdout, owner+"/"+repo, changes)
		}
		if len(changes) > 0 {
			return exitDrift
		}
		return exitOK
	}
	if destination == "" {
		_, err = stdout.Write(b)
		if err != nil {
			return reportError(stderr, err)
		}
		return exitOK
	}
	if err := atomicWrite(destination, b); err != nil {
		return reportError(stderr, err)
	}
	fmt.Fprintf(stderr, "%s %s %s %s\n", errStyle.green(errStyle.bold("Exported")), errStyle.cyan(owner+"/"+repo), errStyle.dim("to"), errStyle.cyan(destination))
	return exitOK
}

func scopeFor(c *config.Config, verbose bool) github.ReadScope {
	selected := c.Actions != nil && c.Actions.SelectedActions != nil
	return github.ReadScope{Repository: c.Repository != nil, Security: c.Security != nil, Actions: c.Actions != nil, SelectedActions: selected, Collaborators: c.Collaborators != nil, Teams: c.Teams != nil, Rulesets: c.Rulesets != nil, Verbose: verbose}
}

func printPlan(w io.Writer, p *reconcile.Plan) {
	printPlanStyled(w, p, styleFor(w))
}

func printPlanStyled(w io.Writer, p *reconcile.Plan, s styler) {
	fmt.Fprintf(w, "%s %s\n\n", s.bold("Repository:"), s.cyan(p.Repository))
	for _, warning := range p.Warnings {
		fmt.Fprintf(w, "%s %s\n", s.yellow(s.bold("Warning:")), warning)
	}
	if len(p.Warnings) > 0 {
		fmt.Fprintln(w)
	}
	if len(p.Changes) == 0 {
		fmt.Fprintln(w, s.green(s.bold("No changes.")))
	} else {
		fmt.Fprintln(w, s.bold("Changes:"))
		for _, c := range p.Changes {
			switch c.Operation {
			case reconcile.Add:
				fmt.Fprintf(w, "\n  %s\n    %s %s\n", s.cyan(c.Path), s.green("add:"), s.green(format(c.After)))
			case reconcile.Remove:
				fmt.Fprintf(w, "\n  %s\n    %s %s\n", s.cyan(c.Path), s.red("remove:"), s.red(format(c.Before)))
			default:
				fmt.Fprintf(w, "\n  %s\n    %s %s %s\n", s.cyan(c.Path), s.yellow(format(c.Before)), s.dim("->"), s.green(format(c.After)))
			}
		}
	}
	if len(p.Unmanaged) > 0 {
		fmt.Fprintf(w, "\n%s %s\n", s.dim("Unmanaged settings:"), s.dim(strings.Join(p.Unmanaged, ", ")))
	}
}
func format(v any) string {
	if v == nil {
		return "null"
	}
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	case bool:
		return fmt.Sprint(x)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}
func confirm(r io.Reader, w io.Writer, n int) bool {
	s := styleFor(w)
	fmt.Fprintf(w, "\n%s %s ", s.yellow(s.bold(fmt.Sprintf("Apply %d change(s)?", n))), s.dim("[y/N]"))
	line, _ := bufio.NewReader(r).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}
func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
func reportError(w io.Writer, err error) int { printError(w, err); return exitError }
func actionableAPIError(err error) error {
	var apiErr *github.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case 401:
			return fmt.Errorf("authentication failed; run 'gh auth login' or set GH_TOKEN/GITHUB_TOKEN: %w", err)
		case 403:
			return fmt.Errorf("GitHub denied access; verify token repository, Administration, Actions, and organization Members permissions: %w", err)
		case 404:
			return fmt.Errorf("repository or managed resource was not found; verify the repository and token access: %w", err)
		}
	}
	return err
}

func resolveToken() (string, string, error) {
	if path, err := exec.LookPath("gh"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, "auth", "token", "--hostname", "github.com")
		if b, err := cmd.Output(); err == nil && strings.TrimSpace(string(b)) != "" {
			return strings.TrimSpace(string(b)), "GitHub CLI credentials", nil
		}
	}
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, name, nil
		}
	}
	return "", "", errors.New("no GitHub credentials found; run 'gh auth login' or set GH_TOKEN or GITHUB_TOKEN")
}

var scpRemote = regexp.MustCompile(`^(?:[^@]+@)?github\.com:([^/]+)/(.+?)(?:\.git)?$`)

func resolveRepo(explicit string) (owner, repo, root string, err error) {
	value := explicit
	if value == "" {
		r, ok := gitRoot()
		if !ok {
			return "", "", "", errors.New("cannot determine repository; use --repo OWNER/REPO or GHREPOCFG_REPO")
		}
		owner, repo, err = repoFromGit(r)
		if err != nil {
			return "", "", "", err
		}
		return owner, repo, r, nil
	}
	if r, ok := gitRoot(); ok {
		explicitOwner, explicitRepo, parseErr := parseRepo(value)
		if parseErr == nil {
			for _, candidate := range gitRemoteRepos(r) {
				if strings.EqualFold(candidate[0], explicitOwner) && strings.EqualFold(candidate[1], explicitRepo) {
					root = r
					break
				}
			}
		}
	}
	owner, repo, err = parseRepo(value)
	return
}

func repoFromGit(root string) (string, string, error) {
	repos := gitRemoteRepos(root)
	if len(repos) == 0 {
		return "", "", errors.New("cannot infer a GitHub.com repository from configured git remotes; use --repo OWNER/REPO")
	}
	if len(repos) > 1 {
		return "", "", errors.New("multiple GitHub.com repositories are configured as git remotes; use --repo OWNER/REPO")
	}
	return repos[0][0], repos[0][1], nil
}

func gitRemoteRepos(root string) [][2]string {
	namesRaw, err := exec.Command("git", "-C", root, "remote").Output()
	if err != nil {
		return nil
	}
	names := strings.Fields(string(namesRaw))
	sort.Strings(names)
	var repos [][2]string
	seen := map[string]bool{}
	for _, name := range names {
		b, err := exec.Command("git", "-C", root, "remote", "get-url", name).Output()
		if err != nil {
			continue
		}
		owner, repo, err := parseRepo(strings.TrimSpace(string(b)))
		if err != nil {
			continue
		}
		key := strings.ToLower(owner + "/" + repo)
		if !seen[key] {
			seen[key] = true
			repos = append(repos, [2]string{owner, repo})
		}
	}
	return repos
}
func parseRepo(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if m := scpRemote.FindStringSubmatch(value); m != nil {
		repo := strings.TrimSuffix(m[2], ".git")
		if repo == "" {
			return "", "", fmt.Errorf("invalid repository %q; expected OWNER/REPO", value)
		}
		return m[1], repo, nil
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil || !strings.EqualFold(u.Hostname(), "github.com") {
			return "", "", fmt.Errorf("repository must be a GitHub.com OWNER/REPO or remote URL")
		}
		value = strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/")
	}
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(value, "?#") {
		return "", "", fmt.Errorf("invalid repository %q; expected OWNER/REPO", value)
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if repo == "" {
		return "", "", fmt.Errorf("invalid repository %q; expected OWNER/REPO", value)
	}
	return parts[0], repo, nil
}

func gitRoot() (string, bool) {
	b, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}

func atomicWrite(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".ghrepocfg-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0644); err == nil {
		_, err = f.Write(b)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

type configChange struct {
	Operation reconcile.Operation `json:"operation"`
	Path      string              `json:"path"`
	Before    any                 `json:"before,omitempty"`
	After     any                 `json:"after,omitempty"`
}

func diffConfig(before, after *config.Config) []configChange {
	var a any = map[string]any{}
	var b any
	if before != nil {
		x, _ := json.Marshal(before)
		_ = json.Unmarshal(x, &a)
	}
	x, _ := json.Marshal(after)
	_ = json.Unmarshal(x, &b)
	var out []configChange
	walkDiff("", a, b, &out)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
func walkDiff(path string, a, b any, out *[]configChange) {
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if a == nil && bok {
		am = map[string]any{}
		aok = true
	}
	if b == nil && aok {
		bm = map[string]any{}
		bok = true
	}
	if aok && bok {
		keys := map[string]bool{}
		for k := range am {
			keys[k] = true
		}
		for k := range bm {
			keys[k] = true
		}
		for k := range keys {
			p := k
			if path != "" {
				p = path + "." + k
			}
			walkDiff(p, am[k], bm[k], out)
		}
		return
	}
	if reflect.DeepEqual(a, b) {
		return
	}
	op := reconcile.Modify
	if a == nil {
		op = reconcile.Add
	} else if b == nil {
		op = reconcile.Remove
	}
	*out = append(*out, configChange{op, path, a, b})
}
func printExportDiff(w io.Writer, repo string, changes []configChange) {
	printExportDiffStyled(w, repo, changes, styleFor(w))
}

func printExportDiffStyled(w io.Writer, repo string, changes []configChange, s styler) {
	fmt.Fprintf(w, "%s %s\n\n", s.bold("Repository:"), s.cyan(repo))
	if len(changes) == 0 {
		fmt.Fprintln(w, s.green(s.bold("Export would make no changes.")))
		return
	}
	fmt.Fprintln(w, s.bold("Export changes:"))
	for _, c := range changes {
		fmt.Fprintf(w, "\n  %s\n    ", s.cyan(c.Path))
		if c.Operation == reconcile.Modify {
			fmt.Fprintf(w, "%s %s %s %s", s.yellow("modify:"), s.yellow(format(c.Before)), s.dim("->"), s.green(format(c.After)))
		} else if c.Operation == reconcile.Add {
			fmt.Fprintf(w, "%s %s", s.green("add:"), s.green(format(c.After)))
		} else {
			fmt.Fprintf(w, "%s %s", s.red("remove:"), s.red(format(c.Before)))
		}
		fmt.Fprintln(w)
	}
}
