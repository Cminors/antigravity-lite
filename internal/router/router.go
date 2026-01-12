package router

import (
	"regexp"
	"strings"
	"sync"

	"antigravity-lite/config"
)

// Router handles model routing and mapping
type Router struct {
	routes    map[string]string
	patterns  []patternRoute
	mu        sync.RWMutex
}

type patternRoute struct {
	pattern *regexp.Regexp
	target  string
}

// NewRouter creates a new router
func NewRouter(cfg *config.Config) *Router {
	r := &Router{
		routes: make(map[string]string),
	}

	// Load routes from config
	for _, route := range cfg.Routes {
		r.AddRoute(route.Pattern, route.Target)
	}

	return r
}

// AddRoute adds a route mapping
func (r *Router) AddRoute(pattern, target string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if it's a wildcard pattern
	if strings.Contains(pattern, "*") {
		// Convert glob to regex
		regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$"
		if re, err := regexp.Compile(regexPattern); err == nil {
			r.patterns = append(r.patterns, patternRoute{
				pattern: re,
				target:  target,
			})
		}
	} else {
		// Exact match
		r.routes[pattern] = target
	}
}

// RemoveRoute removes a route
func (r *Router) RemoveRoute(pattern string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routes, pattern)

	// Remove from patterns
	for i, p := range r.patterns {
		if p.pattern.String() == "^"+strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*")+"$" {
			r.patterns = append(r.patterns[:i], r.patterns[i+1:]...)
			break
		}
	}
}

// Route returns the target model for a given source model
func (r *Router) Route(model string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check exact match first
	if target, ok := r.routes[model]; ok {
		return target
	}

	// Check pattern matches
	for _, p := range r.patterns {
		if p.pattern.MatchString(model) {
			return p.target
		}
	}

	// No match, return original
	return model
}

// GetRoutes returns all routes
func (r *Router) GetRoutes() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make(map[string]string)
	for k, v := range r.routes {
		routes[k] = v
	}

	return routes
}

// SetRoutes replaces all routes
func (r *Router) SetRoutes(routes map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = make(map[string]string)
	r.patterns = nil

	for pattern, target := range routes {
		if strings.Contains(pattern, "*") {
			regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$"
			if re, err := regexp.Compile(regexPattern); err == nil {
				r.patterns = append(r.patterns, patternRoute{
					pattern: re,
					target:  target,
				})
			}
		} else {
			r.routes[pattern] = target
		}
	}
}

// IsBackgroundRequest checks if the request is a background task
func (r *Router) IsBackgroundRequest(messages []map[string]interface{}) bool {
	// Check for common background task patterns
	backgroundPatterns := []string{
		"generate a title",
		"summarize",
		"create a headline",
		"generate title",
	}

	if len(messages) == 0 {
		return false
	}

	lastMsg := messages[len(messages)-1]
	content, ok := lastMsg["content"].(string)
	if !ok {
		return false
	}

	content = strings.ToLower(content)
	for _, pattern := range backgroundPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}

	return false
}

// GetLightModel returns a lightweight model for background tasks
func (r *Router) GetLightModel() string {
	return "gemini-2.0-flash"
}
