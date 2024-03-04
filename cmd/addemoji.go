package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/0supa/degen/client"
	discordClient "github.com/0supa/degen/client/discord"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	regexp "github.com/wasilibs/go-re2"
)

func init() {
	RegisterCommand(Command{
		Name: "addemoji",
		DiscordData: api.CreateCommandData{
			Name:        "addemoji",
			Description: "Steal emojis from other servers",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "emoji",
					Description: "Emojis from other servers that you want to add",
					Required:    true,
				},
			},
			NoDMPermission:           true,
			DefaultMemberPermissions: discord.NewPermissions(discord.PermissionManageEmojisAndStickers),
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			emojiRegexp := regexp.MustCompile(`<a?:(?P<Name>\w+):(?P<ID>\d+)>`)

			var options struct {
				EmojiString string `discord:"emoji"`
			}

			if err := cmd.Data.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			var builder strings.Builder

			emojis := emojiRegexp.FindAllStringSubmatch(options.EmojiString, 6)

			if len(emojis) == 0 {
				return Response("No emojis were found in your string")
			}

			if len(emojis) > 5 {
				builder.WriteString("> ⚠ You can't add more than 5 emojis in a single request\n")
				emojis = emojis[:5]
			}

			for _, emoji := range emojis {
				emojiString := emoji[0]
				alias := emoji[1]
				id := emoji[2]

				ext := "png"
				emojiType := "static"

				if strings.HasPrefix(emojiString, "<a:") {
					ext = "gif"
					emojiType = "animated"
				}

				emojiURL := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s?size=128&quality=lossless", id, ext)
				fileRes, err := client.HTTP.Get(emojiURL)
				if err != nil {
					return ErrorResponse(err)
				}

				if fileRes.StatusCode != http.StatusOK {
					builder.WriteString(fmt.Sprintf("\n❌ Failed fetching %s emoji __:%s:__ %s\n", emojiType, alias, DiscordCodeBlock("", fileRes.Status)))
					continue
				}

				var emojiContent []byte
				emojiContent, err = io.ReadAll(fileRes.Body)
				if err != nil {
					return ErrorResponse(err)
				}

				createdEmoji, err := discordClient.Handler.CreateEmoji(cmd.Event.GuildID, api.CreateEmojiData{
					Name: alias,
					Image: api.Image{
						ContentType: fileRes.Header.Get("content-type"),
						Content:     emojiContent,
					},
				})
				if err != nil {
					builder.WriteString(fmt.Sprintf("\n❌ Failed adding %s emoji __:%s:__ %s\n", emojiType, alias, DiscordCodeBlock("", err.Error())))
					continue
				}

				idPrefix := ""
				if createdEmoji.Animated {
					idPrefix = "a"
				}
				builder.WriteString(fmt.Sprintf("\n✅ Successfully added %s emoji <%s:%s>", emojiType, idPrefix, createdEmoji.APIString()))
			}

			return Response(builder.String())
		},
	})
}
