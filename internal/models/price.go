package models

import (
	"strconv"
	"strings"
	"unicode"
)

type Price struct {
	Base        float64
	Shipping    float64
	Tax         float64
	Discounts   float64
	Total       float64
	Currency    string
	TotalString string
}

func ParsePrice(price string) (float64, string, error) {
	price = strings.TrimSpace(price)

	if price == "" {
		return 0, "", nil
	}

	currency, number := "", ""

	for _, char := range price {
		currency, number = processCharacter(char, currency, number)
	}

	float, err := strconv.ParseFloat(number, 64)

	if err != nil {
		return 0, "", err
	}

	return float, currency, nil
}

func processCharacter(char rune, currency, number string) (string, string) {
	if isSpaceOrPlus(char) {
		return currency, number
	} else if isSeparatorChar(char) {
		number += "."
	} else if unicode.IsDigit(char) {
		number += string(char)
	} else {
		currency += string(char)
	}
	return currency, number
}

func isSeparatorChar(char rune) bool {
	return char == '.' || char == ','
}

func isSpaceOrPlus(char rune) bool {
	return char == ' ' || char == '+'
}
