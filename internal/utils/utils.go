package utils

import (
	"regexp"
	"strings"
)

func EscapeMarkdownV2(text string) string {
	// Создаем карту замен для специальных символов
	replacements := map[string]string{
		"_": "\\_",
		"*": "\\*",
		"[": "\\[",
		"]": "\\]",
		"~": "\\~",
		"`": "\\`",
		">": "\\>",
		"#": "\\#",
		"+": "\\+",
		"-": "\\-",
		"=": "\\=",
		"|": "\\|",
		".": "\\.",
		"!": "\\!",
	}

	// Буфер для построения результата
	var result strings.Builder
	for _, char := range text {
		// Специально убираем экранирование для скобок
		if char == '(' || char == ')' {
			result.WriteRune(char)
			continue
		}

		// Обработка остальных специальных символов
		if replacement, exists := replacements[string(char)]; exists {
			result.WriteString(replacement)
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

func SanitizeMarkdownText(text string) string {
	// Удаляем лишние экранирования
	text = strings.ReplaceAll(text, "\\)", ")")
	text = strings.ReplaceAll(text, "\\(", "(")

	// Заменяем двойные экранирования
	text = regexp.MustCompile(`\\{2,}`).ReplaceAllString(text, "\\")

	return text
}
