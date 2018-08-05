package kmip

import (
	"bytes"
	"github.com/ansel1/merry"
	"reflect"
	"regexp"
	"strings"
)

func Marshal(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unmarshal(b []byte, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return merry.New("non-pointer passed to Unmarshal")
	}
	return unmarshal(val, TTLV(b))
}

// implementation of 5.4.1.1 and 5.5.1.1
func NormalizeName(s string) string {
	// 1. Replace round brackets ‘(‘, ‘)’ with spaces
	s = regexp.MustCompile(`[()]`).ReplaceAllString(s, " ")

	// 2. If a non-word char (not alpha, digit or underscore) is followed by a letter (either upper or lower case) then a lower case letter, replace the non-word char with space
	s = regexp.MustCompile(`(\W)([a-zA-Z][a-z])`).ReplaceAllString(s, " $2")

	words := strings.Split(s, " ")

	for i, w := range words {
		// 3. Replace remaining non-word chars (except whitespace) with underscore.
		w = regexp.MustCompile(`\W`).ReplaceAllString(w, "_")

		if i == 0 {
			// 4. If the first word begins with a digit, move all digits at start of first word to end of first word
			w = regexp.MustCompile(`^([\d]+)(.*)`).ReplaceAllString(w, `$2$1`)
		}

		// 5. Capitalize the first letter of each word
		words[i] = strings.Title(w)
	}

	// 6. Concatenate all words with spaces removed
	return strings.Join(words, "")

}
