package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0supa/whatbot/client"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
)

type nekoResponse struct {
	URL string `json:"url"`
}

var verbs = map[string]string{
	"kiss":   "kisses",
	"hug":    "hugs",
	"pat":    "pats",
	"tickle": "tickles",
	"cuddle": "cuddles with",
}

func init() {
	RegisterCommand(Command{
		Name: "nekos",
		DiscordData: api.CreateCommandData{
			Name:        "nekos",
			Description: "Get a weeb gif for a specific action",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "action",
					Description: "The action you want a gif for",
					Required:    true,
					Choices: []discord.StringChoice{
						{Name: "Kiss", Value: "kiss"},
						{Name: "Hug", Value: "hug"},
						{Name: "Pat", Value: "pat"},
						{Name: "Ticke", Value: "tickle"},
						{Name: "Cuddle", Value: "cuddle"},
					},
				},
				&discord.UserOption{
					OptionName:  "receiver",
					Description: "The user that gets your action",
					Required:    false,
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			var options struct {
				Action string         `discord:"action"`
				Target discord.UserID `discord:"receiver?"`
			}

			if err := cmd.Data.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			res, err := client.HTTP.Get("https://nekos.life/api/v2/img/" + options.Action)
			if err != nil {
				return ErrorResponse(err)
			}

			data := nekoResponse{}

			if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
				return ErrorResponse(err)
			}

			Sender := cmd.Event.Sender()

			var adj = "themself"
			if Sender.ID != options.Target {
				adj = options.Target.Mention()
			}

			var description string
			if options.Target != 0 {
				description = fmt.Sprintf("%s %s %s", Sender.Mention(), verbs[options.Action], adj)
			}

			return &api.InteractionResponseData{
				Embeds: &[]discord.Embed{{
					// Color:       0x3092790,
					Image:       &discord.EmbedImage{URL: data.URL},
					Description: description,
				}},
			}
		},
	})
}
