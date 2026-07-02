package context_sanitizer

import (
	"strings"
	"unicode"
)

// normalizeConfusables 规范化混淆字符
// 设计文档 6.2.1: 增强混淆检测
func normalizeConfusables(s string) string {
	// RTL override 剥离
	s = strings.Map(func(r rune) rune {
		switch r {
		case 0x202e, 0x202d, 0x202a, 0x202b, 0x202c: // RTL/LTR override & embedding
			return -1
		default:
			return r
		}
	}, s)

	// Combining diacritical marks 剥离 (U+0300 - U+036F)
	s = strings.Map(func(r rune) rune {
		if r >= 0x0300 && r <= 0x036f {
			return -1
		}
		return r
	}, s)

	// 全角转半角
	s = strings.Map(fullwidthToHalfwidth, s)

	// Unicode 同形字映射 (常见攻击场景)
	s = mapHomoglyphs(s)

	return s
}

// fullwidthToHalfwidth 全角字符转半角
func fullwidthToHalfwidth(r rune) rune {
	// 全角 ASCII (U+FF01 - U+FF5E) -> 半角 (U+0021 - U+007E)
	if r >= 0xff01 && r <= 0xff5e {
		return r - 0xfee0
	}
	// 全角空格
	if r == 0x3000 {
		return ' '
	}
	return r
}

// mapHomoglyphs 映射常见的 Unicode 同形字
func mapHomoglyphs(s string) string {
	// Cyrillic -> Latin 映射表 (常见攻击字符)
	replacements := map[rune]rune{
		// Cyrillic lowercase
		0x0430: 'a', // а -> a
		0x0435: 'e', // е -> e
		0x043e: 'o', // о -> o
		0x0440: 'p', // р -> p
		0x0441: 'c', // с -> c
		0x0445: 'x', // х -> x
		0x0455: 's', // ѕ -> s
		0x0456: 'i', // і -> i
		0x0458: 'j', // ј -> j
		0x04cf: 'l', // ӏ -> l

		// Cyrillic uppercase
		0x0410: 'A', // А -> A
		0x0412: 'B', // В -> B
		0x0415: 'E', // Е -> E
		0x041a: 'K', // К -> K
		0x041c: 'M', // М -> M
		0x041d: 'H', // Н -> H
		0x041e: 'O', // О -> O
		0x0420: 'P', // Р -> P
		0x0421: 'C', // С -> C
		0x0422: 'T', // Т -> T
		0x0425: 'X', // Х -> X

		// Greek -> Latin
		0x03bf: 'o', // ο -> o
		0x03c1: 'p', // ρ -> p
		0x03c5: 'v', // υ -> v
		0x03c9: 'w', // ω -> w
		0x0391: 'A', // Α -> A
		0x0392: 'B', // Β -> B
		0x0395: 'E', // Ε -> E
		0x0396: 'Z', // Ζ -> Z
		0x0397: 'H', // Η -> H
		0x0399: 'I', // Ι -> I
		0x039a: 'K', // Κ -> K
		0x039c: 'M', // Μ -> M
		0x039d: 'N', // Ν -> N
		0x039f: 'O', // Ο -> O
		0x03a1: 'P', // Ρ -> P
		0x03a4: 'T', // Τ -> T
		0x03a5: 'Y', // Υ -> Y
		0x03a7: 'X', // Χ -> X
	}

	return strings.Map(func(r rune) rune {
		if mapped, ok := replacements[r]; ok {
			return mapped
		}
		return r
	}, s)
}

// detectObfuscationLayer 检测混淆层数并返回得分
func detectObfuscationLayer(original, normalized string) int {
	if original == normalized {
		return 0
	}

	score := 0

	// RTL override 检测
	if containsRTL(original) {
		score += 30
	}

	// Combining marks 检测
	if containsCombiningMarks(original) {
		score += 25
	}

	// 全角字符检测
	if containsFullwidth(original) {
		score += 20
	}

	// 同形字检测
	if containsHomoglyphs(original) {
		score += 35
	}

	return score
}

func containsRTL(s string) bool {
	for _, r := range s {
		if r >= 0x202a && r <= 0x202e {
			return true
		}
	}
	return false
}

func containsCombiningMarks(s string) bool {
	for _, r := range s {
		if r >= 0x0300 && r <= 0x036f {
			return true
		}
	}
	return false
}

func containsFullwidth(s string) bool {
	for _, r := range s {
		if (r >= 0xff01 && r <= 0xff5e) || r == 0x3000 {
			return true
		}
	}
	return false
}

func containsHomoglyphs(s string) bool {
	for _, r := range s {
		// Cyrillic 范围
		if (r >= 0x0400 && r <= 0x04ff) && unicode.IsLetter(r) {
			return true
		}
		// Greek 范围
		if (r >= 0x0370 && r <= 0x03ff) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// normalizeWhitespace 规范化空白字符
func normalizeWhitespace(s string) string {
	// 各种空白字符统一为普通空格
	whitespaceChars := map[rune]rune{
		0x00a0: ' ', // Non-breaking space
		0x1680: ' ', // Ogham space mark
		0x2000: ' ', // En quad
		0x2001: ' ', // Em quad
		0x2002: ' ', // En space
		0x2003: ' ', // Em space
		0x2004: ' ', // Three-per-em space
		0x2005: ' ', // Four-per-em space
		0x2006: ' ', // Six-per-em space
		0x2007: ' ', // Figure space
		0x2008: ' ', // Punctuation space
		0x2009: ' ', // Thin space
		0x200a: ' ', // Hair space
		0x202f: ' ', // Narrow no-break space
		0x205f: ' ', // Medium mathematical space
		0x3000: ' ', // Ideographic space
	}

	return strings.Map(func(r rune) rune {
		if mapped, ok := whitespaceChars[r]; ok {
			return mapped
		}
		return r
	}, s)
}
