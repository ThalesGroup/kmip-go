package kmiputil

import (
	"regexp"
	"strings"
)

var nonWordAtWordBoundary = regexp.MustCompile(`(\W)([a-zA-Z][a-z])`)
var startingDigits = regexp.MustCompile(`^([\d]+)(.*)`)

// implementation of 5.4.1.1 and 5.5.1.1
func NormalizeName(s string) string {
	// 1. Replace round brackets ‘(‘, ‘)’ with spaces
	s = strings.Map(func(r rune) rune {
		switch r {
		case '(', ')':
			return ' '
		}
		return r
	}, s)

	// 2. If a non-word char (not alpha, digit or underscore) is followed by a letter (either upper or lower case) then a lower case letter, replace the non-word char with space
	s = nonWordAtWordBoundary.ReplaceAllString(s, " $2")

	// 3. Replace remaining non-word chars (except whitespace) with underscore.
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		case r == ' ':
		default:
			return '_'
		}
		return r
	}, s)

	words := strings.Split(s, " ")

	for i, w := range words {

		if i == 0 {
			// 4. If the first word begins with a digit, move all digits at start of first word to end of first word
			w = startingDigits.ReplaceAllString(w, `$2$1`)
		}

		// 5. Capitalize the first letter of each word
		words[i] = strings.Title(w)
	}

	// 6. Concatenate all words with spaces removed
	return strings.Join(words, "")

}
