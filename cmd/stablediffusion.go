package cmd

import (
	"context"

	"github.com/0supa/whatbot/client"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

func init() {
	RegisterCommand(Command{
		Name: "stable-diffusion",
		DiscordData: api.CreateCommandData{
			Name:        "stable-diffusion",
			Description: "Run a Stable Diffusion Text-to-Image AI prompt",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "prompt",
					Description: "Text-to-Image AI Prompt",
					Required:    true,
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			var options struct {
				Prompt string `discord:"prompt"`
			}

			if err := cmd.Data.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			body, err := client.StableDiffusionImage(options.Prompt)
			if err != nil {
				return ErrorResponse(err)
			}

			return &api.InteractionResponseData{
				Content: option.NewNullableString("> " + options.Prompt),
				Files: []sendpart.File{{
					Name:   "image.png",
					Reader: body,
				}},
			}
		},
	})
}
