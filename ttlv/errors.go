package ttlv

import "github.com/ansel1/merry"

func Is(err error, originals ...error) bool {
	return merry.Is(err, originals...)
}

func Details(err error) string {
	return merry.Details(err)
}
