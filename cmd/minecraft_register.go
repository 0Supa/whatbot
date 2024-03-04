package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	discordClient "github.com/0supa/degen/client/discord"
	"github.com/0supa/degen/client/pwd"
	"github.com/0supa/degen/client/sql"
	"github.com/0supa/degen/client/store"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var userKeys = make(map[discord.UserID]*store.Key)

func init() {
	guilds := []discord.GuildID{
		1200915706661843074, // dulas
		761682825439084544,  // sus
		776226518086451230,  // xd
	}

	bannedUsers := []discord.UserID{
		486605322924982284, // eduart.pxx
	}

	channelIDs := []discord.ChannelID{
		1206783236387250266, // #mc-registration
		797616247918821436,  // #testing
	}

	RegisterCommand(Command{
		Name:   "minecraft",
		Guilds: guilds,
		DiscordData: api.CreateCommandData{
			Name:        "minecraft",
			Description: "Minecraft SMP command",
			Options: []discord.CommandOption{
				&discord.SubcommandOption{
					OptionName:  "register",
					Description: "Register a new account on the SMP server!",
				},
			},
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			if !slices.Contains(guilds, cmd.Event.GuildID) {
				return &api.InteractionResponseData{
					Content: option.NewNullableString("You aren't allowed to run this command in this server."),
					Flags:   discord.EphemeralMessage,
				}
			}

			discordUID := cmd.Event.SenderID()

			if slices.Contains(bannedUsers, discordUID) {
				return Response("Your account is prohibited from using this command.")
			}

			if !slices.Contains(channelIDs, cmd.Event.ChannelID) {
				return &api.InteractionResponseData{
					Content: option.NewNullableString("Please use this command in <#1206783236387250266>."),
					Flags:   discord.EphemeralMessage,
				}
			}

			if userKeys[discordUID] != nil && time.Now().Before(userKeys[discordUID].Expiry) {
				return &api.InteractionResponseData{
					Content: option.NewNullableString("Please wait some time before making a request again."),
					Flags:   discord.EphemeralMessage,
				}
			}

			player, err := sql.GetPlayer("", discordUID.String())
			if err == nil {
				return Response("%s, you are already registered on the server.", player.Name)
			}
			if err != sql.ErrNil {
				return ErrorResponse(err)
			}

			salt := pwd.GenerateSalt(16)
			hash := hex.EncodeToString(salt)

			key := &store.Key{
				Hash:   hash,
				User:   *cmd.Event.Sender(),
				Expiry: time.Now().Add(30 * time.Minute),
			}

			store.RegisterKeys[hash] = key
			userKeys[discordUID] = key

			dm, err := discordClient.Handler.CreatePrivateChannel(discordUID)
			if err != nil {
				return ErrorResponse(err)
			}

			_, err = discordClient.Handler.State.SendMessage(dm.ID, fmt.Sprintf("Please [**click here**](https://dulas.supa.sh/mc/register/?key=%s) in order to register.\nDo not share this link with anyone!", hash))
			if err != nil {
				return ErrorResponse(err)
			}

			return Response("You've been messaged a registration link!")
		},
	})
}
