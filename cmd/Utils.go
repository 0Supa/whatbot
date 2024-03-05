package cmd

import "strings"

func CodeBlock(lang string, s string) string {
	return "```" + lang + "\n" + strings.ReplaceAll(s, "`", "`\u200B") + "```"
}
