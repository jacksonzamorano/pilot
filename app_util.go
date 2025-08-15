package pilot

// PathListFromString parses a URL path into individual path components for routing.
// This utility function splits a URL path by forward slashes and returns the components
// as a slice, enabling efficient path matching in the routing trie structure.
//
// The function handles edge cases like:
//   - Leading forward slash removal
//   - Empty path components (consecutive slashes)
//   - Root path ("/") handling
//   - Trailing slash normalization
//
// Parameters:
//   - path: URL path string to split (e.g., "/api/users/123")
//
// Returns:
//   - []string: Slice of path components without leading/trailing slashes
//
// Examples:
//   - "/api/users/123" → ["api", "users", "123"]
//   - "/users" → ["users"]
//   - "/" → [""] (single empty component)
//   - "/api/users/" → ["api", "users"]
//
// This function is used internally by the routing system to convert URL paths
// into components that can be matched against the routing trie structure.
// Each component becomes a node in the trie, enabling efficient O(path_length)
// route lookups regardless of the total number of registered routes.
func PathListFromString(path string) []string {
	route := []string{}
	start := 1
	end := 1
	for end < len(path) {
		if path[end] == '/' {
			route = append(route, path[start:end])
			start = end + 1
		}
		end++
	}
	if start != end || start == 1 {
		route = append(route, path[start:end])
	}
	return route
}
