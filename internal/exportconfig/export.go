// Package exportconfig maps GitHub state to full or scope-preserving YAML.
package exportconfig

import (
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"github.com/UnitVectorY-Labs/ghrepocfg/internal/github"
	"strings"
)

func FromState(s *github.State) *config.Config {
	c := s.Additional
	c.Repository, c.Security, c.Actions = s.Repository, s.Security, s.Actions
	properties := cloneCustomProperties(s.CustomProperties)
	c.CustomProperties = &properties
	collabs := map[string]config.Access{}
	for name, v := range s.Collaborators {
		collabs[name] = config.Access{Permission: v.Permission}
	}
	c.Collaborators = &collabs
	teams := map[string]config.Access{}
	for name, v := range s.Teams {
		teams[name] = config.Access{Permission: v.Permission}
	}
	c.Teams = &teams
	rules := map[string]config.Ruleset{}
	for name, v := range s.Rulesets {
		rules[name] = v.Value
	}
	c.Rulesets = &rules
	return &c
}

func ScopedFromState(base *config.Config, s *github.State) *config.Config {
	out := config.Project(base, FromState(s))
	if base.Environments != nil && out.Environments != nil {
		environments := make(map[string]config.Environment, len(*out.Environments))
		for name, live := range *out.Environments {
			environments[name] = live
			for managedName, managed := range *base.Environments {
				if strings.EqualFold(managedName, name) {
					environments[name] = *config.Project(&managed, &live)
					break
				}
			}
		}
		out.Environments = &environments
	}
	return out
}

func cloneCustomProperties(in map[string]config.CustomPropertyValue) map[string]config.CustomPropertyValue {
	out := make(map[string]config.CustomPropertyValue, len(in))
	for name, value := range in {
		if list, ok := value.Value.([]string); ok {
			value.Value = append([]string(nil), list...)
		}
		out[name] = value
	}
	return out
}
