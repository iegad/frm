package utils

import (
	"math/rand"
)

var (
	normalChars = [62]byte{
		'1', '2', '3', '4', '5', '6', '7', '8', '9', '0',
		'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p',
		'Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P',
		'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l',
		'A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L',
		'z', 'x', 'c', 'v', 'b', 'n', 'm',
		'Z', 'X', 'C', 'V', 'B', 'N', 'M',
	}

	specialChars = [93]byte{
		'1', '2', '3', '4', '5', '6', '7', '8', '9', '0',
		'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p',
		'Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P',
		'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l',
		'A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L',
		'z', 'x', 'c', 'v', 'b', 'n', 'm',
		'Z', 'X', 'C', 'V', 'B', 'N', 'M',
		'`', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '-', '=', '~', '_', '+',
		'[', ']', '{', '}', '|',
		';', ':', '"',
		',', '.', '/', '<', '>', '?',
	}
)

func RandPassword(n int, specialSymbol bool) string {
	ret := [16]byte{}

	if specialSymbol {
		for i := 0; i < n; i++ {
			ret[i] = specialChars[rand.Int31n(93)]
		}
	} else {
		for i := 0; i < n; i++ {
			ret[i] = normalChars[rand.Int31n(62)]
		}
	}

	return string(ret[:])
}
