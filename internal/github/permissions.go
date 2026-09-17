package github

import "strings"

// RequiredPermission is a diagnostic hint, not a claim about granted access.
// Response X-Accepted-GitHub-Permissions takes precedence over this fallback.
func RequiredPermission(method, path string) string {
	read := method == "GET"
	level := "write"
	if read {
		level = "read"
	}
	permission := "Administration"
	switch {
	case strings.Contains(path, "/properties/values"):
		permission = "Custom properties"
		if read {
			permission = "Metadata"
		}
	case strings.Contains(path, "/actions/variables"):
		permission = "Variables"
	case strings.Contains(path, "/environments/") && strings.Contains(path, "/variables"):
		permission = "Environments"
	case strings.Contains(path, "/actions/oidc/"):
		permission = "Actions"
	case strings.Contains(path, "/actions/cache/storage-limit") && read:
		permission = "Actions"
	case strings.Contains(path, "/environments") && read:
		permission = "Actions"
	case strings.HasSuffix(path, "/pages"):
		if !read {
			return "Pages: write and Administration: write"
		}
		permission = "Pages"
	case strings.Contains(path, "/labels"):
		return "Issues: " + level + " or Pull requests: " + level
	case strings.HasPrefix(path, "/orgs/") && strings.Contains(path, "/teams/"):
		return "Administration: write and organization Members: read"
	case read && (strings.Contains(path, "/rulesets") || strings.HasSuffix(path, "/private-vulnerability-reporting") || strings.Contains(path, "/collaborators")):
		permission = "Metadata"
	case read && strings.Count(strings.Trim(path, "/"), "/") == 2:
		permission = "Metadata"
	}
	if path == "" {
		return ""
	}
	return permission + ": " + level
}
