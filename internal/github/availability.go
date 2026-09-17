package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
)

// Unavailable describes state that cannot safely be exported or reconciled.
type Unavailable struct {
	Path   string   `json:"path"`
	Reason string   `json:"reason"`
	Parts  []string `json:"-"`
}

func (s *State) Omit(parts []string, reason string) {
	path := strings.Join(parts, ".")
	for _, u := range s.Unavailable {
		if u.Path == path {
			return
		}
	}
	s.Unavailable = append(s.Unavailable, Unavailable{Path: path, Reason: reason, Parts: parts})
}
func (s *State) UnavailablePaths() [][]string {
	paths := make([][]string, 0, len(s.Unavailable))
	for _, u := range s.Unavailable {
		paths = append(paths, u.Parts)
	}
	return paths
}
func (s *State) Available(c *config.Config) *config.Config {
	return config.WithUnavailable(c, nil, s.UnavailablePaths())
}

// Read failures are recoverable only for access denial or ambiguous absence.
// A failed baseline repository read is always fatal and never passes here.
func (s *State) read(paths []string, fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}
	reason, ok := UnavailableReason(err, true)
	if !ok {
		return fmt.Errorf("read %s: %w", strings.Join(paths, ", "), err)
	}
	for _, path := range paths {
		s.Omit(strings.Split(path, "."), reason)
	}
	return nil
}

func IsRateLimit(err error) bool {
	var e *APIError
	if !errors.As(err, &e) {
		return false
	}
	message := strings.ToLower(e.Message)
	return e.Status == 429 || (e.Status == 403 && (e.Headers.Get("X-RateLimit-Remaining") == "0" || e.Headers.Get("Retry-After") != "" || strings.Contains(message, "rate limit") || strings.Contains(message, "abuse detection")))
}

// UnavailableReason never downgrades a partially completed mutation.
type unreadableState struct{ reason string }

func (e *unreadableState) Error() string { return e.reason }

func UnavailableReason(err error, reading bool) (string, bool) {
	var unknown *unreadableState
	if reading && errors.As(err, &unknown) {
		return unknown.reason, true
	}
	var partial *PartialError
	if errors.As(err, &partial) {
		return "", false
	}
	var e *APIError
	if !errors.As(err, &e) || IsRateLimit(err) {
		return "", false
	}
	var reason string
	switch e.Status {
	case http.StatusForbidden:
		reason = "GitHub denied access (403)"
	case http.StatusNotFound:
		if !reading {
			return "", false
		}
		reason = "resource unavailable or inaccessible (404)"
	default:
		return "", false
	}
	if e.Message != "" {
		reason += "; " + e.Message
	}
	p := e.Headers.Get("X-Accepted-GitHub-Permissions")
	if p == "" {
		p = RequiredPermission(e.Method, e.Path)
	}
	if p != "" {
		reason += "; requires " + p
	}
	return reason, true
}

// PartialError records a failure after a mutation has already succeeded.
type PartialError struct{ Err error }

func (e *PartialError) Error() string { return "partially applied; " + e.Err.Error() }
func (e *PartialError) Unwrap() error { return e.Err }
func AfterMutation(err error) error {
	if err == nil {
		return nil
	}
	return &PartialError{Err: err}
}
