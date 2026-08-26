package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/reconcile"
)

func TestNoColorConvention(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !noColorRequested() {
		t.Fatal("non-empty NO_COLOR must disable color")
	}
	t.Setenv("NO_COLOR", "")
	if noColorRequested() {
		t.Fatal("empty NO_COLOR must not disable color")
	}
}

func TestStyler(t *testing.T) {
	if got := (styler{}).green("ok"); got != "ok" {
		t.Fatalf("disabled style = %q", got)
	}
	got := (styler{enabled: true}).green("ok")
	if got != ansiGreen+"ok"+ansiReset {
		t.Fatalf("enabled style = %q", got)
	}
}

func TestColoredPlanUsesSemanticColors(t *testing.T) {
	p := &reconcile.Plan{
		Repository: "acme/widgets",
		Drift:      true,
		Warnings:   []string{"legacy protection exists"},
		Changes: []reconcile.Change{
			{Operation: reconcile.Add, Path: "collaborators.alice", After: "push"},
			{Operation: reconcile.Modify, Path: "repository.has_wiki", Before: true, After: false},
			{Operation: reconcile.Remove, Path: "teams.old", Before: "pull"},
		},
	}
	var out bytes.Buffer
	printPlanStyled(&out, p, styler{enabled: true})
	got := out.String()
	for _, code := range []string{ansiCyan, ansiGreen, ansiYellow, ansiRed, ansiBold, ansiDim} {
		if !strings.Contains(got, code) {
			t.Errorf("colored plan is missing ANSI code %q: %q", code, got)
		}
	}

	out.Reset()
	printPlanStyled(&out, p, styler{})
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("plain plan contains ANSI escapes: %q", out.String())
	}
}

func TestJSONOutputNeverUsesColor(t *testing.T) {
	var out bytes.Buffer
	writeJSON(&out, map[string]any{"status": "ok"})
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("JSON contains ANSI escapes: %q", out.String())
	}
}
