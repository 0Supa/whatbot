package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0supa/degen/client"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
)

var sfwCategories = []string{"waifu", "neko", "bully", "cuddle", "cry", "hug", "kiss", "lick", "pat", "smug", "bonk", "blush", "smile", "wave", "highfive", "handhold", "nom", "bite", "slap", "kill", "happy", "wink", "poke", "dance", "cringe"}

var nsfwCategories = []string{"waifu", "neko", "trap", "blowjob"}

func init() {
	var sfwChoices []discord.StringChoice
	var nsfwChoices []discord.StringChoice

	for _, v := range sfwCategories {
		sfwChoices = append(sfwChoices, discord.StringChoice{Name: v, Value: v})
	}
	for _, v := range nsfwCategories {
		nsfwChoices = append(nsfwChoices, discord.StringChoice{Name: v, Value: v})
	}

	RegisterCommand(Command{
		Name: "waifupics",
		DiscordData: api.CreateCommandData{
			Name:        "waifupics",
			Description: "Get a waifu.pics image for your chosen category",
			Options: []discord.CommandOption{
				&discord.SubcommandOption{
					OptionName:  "sfw",
					Description: "waifu.pics SFW images",
					Options: []discord.CommandOptionValue{
						&discord.StringOption{
							OptionName:  "category",
							Description: "SFW image category",
							Required:    true,
							Choices:     sfwChoices,
						},
					},
				},
				&discord.SubcommandOption{
					OptionName:  "nsfw",
					Description: "waifu.pics NSFW images",
					Options: []discord.CommandOptionValue{
						&discord.StringOption{
							OptionName:  "category",
							Description: "NSFW image category",
							Required:    true,
							Choices:     nsfwChoices,
						},
					},
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			imgType := cmd.CommandInteractionOption.Name

			if imgType == "nsfw" && !cmd.Event.Channel.NSFW {
				return Response("You can only use this subcommand in NSFW channels")
			}

			var options struct {
				Category string `discord:"category"`
			}

			if err := cmd.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			res, err := client.HTTP.Get(fmt.Sprintf("https://api.waifu.pics/%s/%s", imgType, options.Category))
			if err != nil {
				return ErrorResponse(err)
			}

			data := nekoResponse{}

			if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
				return ErrorResponse(err)
			}

			return &api.InteractionResponseData{
				Embeds: &[]discord.Embed{{
					Image: &discord.EmbedImage{URL: data.URL},
				}},
			}
		},
	})
}
