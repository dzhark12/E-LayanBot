package utils

import (
	"regexp"
	"strings"
)

var requestCodePattern = regexp.MustCompile(`(?i)[a-z]{2}\s*[0-9]{5}`)

func RequestCode(text string) string {
	match := requestCodePattern.FindString(text)
	return strings.ToUpper(strings.Join(strings.Fields(match), ""))
}
