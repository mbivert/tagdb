package main

import (
	"strconv"
)

func xInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return -1
	}

	return i
}

func xItoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Qtag operators
func isop(r byte) bool { return r == '/' || r == '!' /* || r == '|' */ }

// next index which is not an operator if any
func skipOp(s []byte) (i int) {
	for i = 0; i < len(s) && isop(s[i]); i++ {}
	return
}

// next index which is an operator if any
func skipNotOp(s []byte) (i int) {
	for i = 0; i < len(s) && !isop(s[i]); i++ {}
	return
}
