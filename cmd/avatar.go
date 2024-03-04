package cmd

import (
	"context"

	discordClient "github.com/0supa/degen/client/discord"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
)

func init() {
	RegisterCommand(Command{
		Name: "avatar",
		DiscordData: api.CreateCommandData{
			Name:        "avatar",
			Description: "Get a user's avatar",
			Options: []discord.CommandOption{
				&discord.UserOption{
					OptionName:  "target",
					Description: "The user you want to view the avatar from",
					Required:    true,
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			var options struct {
				Target discord.UserID `discord:"target"`
			}

			if err := cmd.Data.Options.Unmarshal(&options); err != nil {
				return ErrorResponse(err)
			}

			embeds := []discord.Embed{}

			User, err := discordClient.Handler.State.User(options.Target)
			if err != nil {
				return ErrorResponse(err)
			}
			embeds = append(embeds, discord.Embed{
				Description: User.Mention() + "'s user avatar",
				Image: &discord.EmbedImage{
					URL: User.AvatarURL() + "?size=4096",
				},
			})

			Member, _ := discordClient.Handler.State.Member(cmd.Event.GuildID, options.Target)
			if Member != nil {
				if guildAvatar := Member.AvatarURL(cmd.Event.GuildID); guildAvatar != "" {
					embeds = append(embeds, discord.Embed{
						Description: User.Mention() + "'s server avatar",
						Image: &discord.EmbedImage{
							URL: guildAvatar + "?size=4096",
						},
					})

				}
			}

			return &api.InteractionResponseData{
				Embeds: &embeds,
			}
		},
	})
}
