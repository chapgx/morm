package morm

var keywords = map[string]bool{
	"primary": true,
}

// safe_keyword makes a field that is a keyword, SQL safe
func safe_keyword(field string) string {
	_, exists := keywords[field]
	if exists {
		return "[" + field + "]"
	}
	return field
}
