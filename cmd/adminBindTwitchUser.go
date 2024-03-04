package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/0supa/degen-go/client"
	discordClient "github.com/0supa/degen-go/client/discord"
	"github.com/0supa/degen-go/client/sql"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/redis/go-redis/v9"
)

func init() {
	RegisterCommand(Command{
		Name: "admin",
		DiscordData: api.CreateCommandData{
			Name:        "admin",
			Description: "Admin commands",
			Options: []discord.CommandOption{
				&discord.SubcommandGroupOption{
					OptionName:  "bind",
					Description: "Binds a <platform> user to a Discord member",
					Subcommands: []*discord.SubcommandOption{
						{
							OptionName:  "twitch",
							Description: "Binds a Twitch user to a Discord member",
							Options: []discord.CommandOptionValue{
								&discord.UserOption{
									OptionName:  "discord-target",
									Description: "The Discord member you want to bind to",
									Required:    true,
								},
								&discord.StringOption{
									OptionName:  "twitch-bind",
									Description: "The Twitch user's name you want to be bound",
									Required:    true,
								},
							},
						},
					},
				},
				&discord.SubcommandGroupOption{
					OptionName:  "unbind",
					Description: "Unbinds a Discord member from <platform>",
					Subcommands: []*discord.SubcommandOption{
						{
							OptionName:  "twitch",
							Description: "Unbinds a Discord member from their Twitch user",
							Options: []discord.CommandOptionValue{
								&discord.UserOption{
									OptionName:  "discord-target",
									Description: "The Discord member you want to unbind",
									Required:    true,
								},
							},
						},
					},
				},
				&discord.SubcommandOption{
					OptionName:  "check-binds",
					Description: "Checks a Discord member's bindings",
					Options: []discord.CommandOptionValue{
						&discord.UserOption{
							OptionName:  "discord-target",
							Description: "The Discord member you want to check bindings for",
							Required:    true,
						},
					},
				},
			},
			NoDMPermission:           true,
			DefaultMemberPermissions: discord.NewPermissions(discord.PermissionAdministrator),
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			cmdOptions := cmd.CommandInteractionOption.Options
			if len(cmdOptions) == 0 {
				return Response("Subcommand options not found")
			}

			guildKey := fmt.Sprintf("%s:DTbinds", cmd.Event.GuildID.String())

			switch cmd.CommandInteractionOption.Name {
			case "bind":
				var options struct {
					Target     discord.UserID `discord:"discord-target"`
					TwitchName string         `discord:"twitch-bind"`
				}

				if err := cmdOptions[0].Options.Unmarshal(&options); err != nil {
					return ErrorResponse(err)
				}

				keyID := options.Target.String()

				if res := client.RedisDB.HExists(ctx, guildKey, keyID); res.Val() {
					return Response("User key already exists, please use unbind first")
				}

				twitchUser, err := client.GetTwitchUser(options.TwitchName, "")
				if err != nil {
					return ErrorResponse(err)
				}

				if twitchUser.ID == "" {
					return Response("Twitch user not found")
				}

				if res := client.RedisDB.HSet(ctx, guildKey, keyID, twitchUser.ID); res != nil {
					if err := res.Err(); err != nil {
						return ErrorResponse(err)
					}
				}

				roles, err := discordClient.Handler.Roles(cmd.Event.GuildID)
				if err != nil {
					return ErrorResponse(err)
				}

				var roleID discord.RoleID
				for _, r := range roles {
					if strings.ToLower(r.Name) == "verified" {
						roleID = r.ID
						break
					}
				}

				if roleID != 0 {
					discordClient.Handler.AddRole(cmd.Event.GuildID, options.Target, roleID, api.AddRoleData{})
				}

				return Response("Successfully bound Twitch user **%s** `%s` to Discord member **%s** `%s`",
					twitchUser.Login, twitchUser.ID, options.Target.Mention(), keyID)
			case "unbind":
				var options struct {
					Target discord.UserID `discord:"discord-target"`
				}

				if err := cmdOptions[0].Options.Unmarshal(&options); err != nil {
					return ErrorResponse(err)
				}

				keyID := options.Target.String()

				res, err := client.RedisDB.HGet(ctx, guildKey, keyID).Result()
				if err := err; err != nil {
					if err == redis.Nil {
						return Response("%s doesn't have a Twitch account bound", options.Target.Mention())
					}

					return ErrorResponse(err)
				}

				deleted, _ := client.RedisDB.HDel(ctx, guildKey, keyID).Result()
				if deleted == 0 {
					return Response("Failed deleting user key")
				}

				return Response("Successfully deleted user key `%s`: `%s`", keyID, res)
			case "check-binds":
				var options struct {
					Target discord.UserID `discord:"discord-target"`
				}

				if err := cmd.Options.Unmarshal(&options); err != nil {
					return ErrorResponse(err)
				}

				var builder strings.Builder
				builder.WriteString(options.Target.Mention() + ":")

				discordUID := options.Target.String()

				userID, err := client.RedisDB.HGet(ctx, guildKey, discordUID).Result()
				if err != nil {
					if err != redis.Nil {
						return ErrorResponse(err)
					}

					builder.WriteString("\n- Twitch: nil")
				} else {
					twitchUser, err := client.GetTwitchUser("", userID)
					if err != nil {
						return ErrorResponse(err)
					}

					builder.WriteString(fmt.Sprintf("\n- Twitch: login=`%s`, id=`%s`", twitchUser.Login, twitchUser.ID))
				}

				player, err := sql.GetPlayer("", discordUID)
				if err != nil {
					if err != sql.ErrNil {
						return ErrorResponse(err)
					}
				} else {
					builder.WriteString(fmt.Sprintf("\n- Minecraft: name=`%s`, uuid=`%s`", player.Name, player.UUID))
				}
				return Response(builder.String())
			}

			return Response("Unknown option")
		},
	})
}
