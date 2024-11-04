package utils

import "unicode"

func CheckLuhn(number string) bool {
	var sum int
	double := false

	for i := len(number) - 1; i >= 0; i-- {
		n := rune(number[i])
		if !unicode.IsDigit(n) {
			return false
		}
		digit := int(n - '0')
		if double {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}
