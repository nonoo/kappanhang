package main

import "strings"

// Checks if all bytes are zeros
func isAllZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func parseNullTerminatedString(d []byte) (res string) {
	nullIndex := strings.Index(string(d), "\x00")
	if nullIndex > 0 {
		res = string(d[:nullIndex])
	}
	return
}
