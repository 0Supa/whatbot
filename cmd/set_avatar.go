package cmd

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/0supa/degen-go/client"
	discordClient "github.com/0supa/degen-go/client/discord"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
)

func init() {
	RegisterCommand(Command{
		Name:   "set-avatar",
		Guilds: []discord.GuildID{761682825439084544},
		DiscordData: api.CreateCommandData{
			Name:        "set-avatar",
			Description: "Set the bot's avatar",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "url",
					Description: "The image URL",
					Required:    true,
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			if cmd.Event.SenderID() != 535820575868715008 {
				return Response("no access")
			}

			var options struct {
				ImgURL string `discord:"url"`
			}

			if err := cmd.Data.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			fileRes, err := client.HTTP.Get(options.ImgURL)
			if err != nil {
				return ErrorResponse(err)
			}

			if fileRes.StatusCode != http.StatusOK {
				return ErrorResponse(errors.New("failed fetching image"))
			}

			var imgData []byte
			imgData, err = io.ReadAll(fileRes.Body)
			if err != nil {
				return ErrorResponse(err)
			}

			_, err = discordClient.Handler.ModifyCurrentUser(api.ModifyCurrentUserData{
				// Username: option.NewString("asd"),
				Avatar: &api.Image{
					Content: imgData,
				},
			})

			if err != nil {
				return ErrorResponse(err)
			}

			return Response("success")
		},
	})
}
