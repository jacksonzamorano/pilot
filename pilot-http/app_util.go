package pilot_http

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
