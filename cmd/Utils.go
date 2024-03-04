package cmd

import "strings"

func DiscordCodeBlock(lang string, s string) string {
	return "```" + lang + "\n" + strings.ReplaceAll(s, "`", "`\u200B") + "```"
}
