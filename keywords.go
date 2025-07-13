package morm

var keywords = map[string]bool{
	"primary": true,
}

func check_keyword(field string) string {
	_, exists := keywords[field]
	if exists {
		return "[" + field + "]"
	}
	return field
}
