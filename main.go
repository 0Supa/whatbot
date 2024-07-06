package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/0supa/degen/client"
	discordClient "github.com/0supa/degen/client/discord"
	_ "github.com/0supa/degen/client/http_server"
	"github.com/0supa/degen/cmd"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/redis/go-redis/v9"
)

var commands []api.CreateCommandData

type banCache struct {
	expiry time.Time
	*client.TwitchBanStatusResponse
}

var banStatusCache = map[string]banCache{}

func main() {
	h := discordClient.Handler

	h.State.AddInteractionHandler(h)
	h.State.AddIntents(gateway.IntentGuilds)
	h.State.AddIntents(gateway.IntentGuildMessages)

	h.State.AddHandler(func(*gateway.ReadyEvent) {
		me, _ := h.State.Me()
		log.Println("connected to the gateway as", me.Tag())

		if err := h.State.Gateway().Send(context.Background(), &gateway.UpdatePresenceCommand{
			Status: discord.IdleStatus,
		}); err != nil {
			log.Println(err)
		}

		for name, command := range cmd.CommandMap {
			if len(command.Guilds) > 0 {
				for _, guildID := range command.Guilds {
					_, err := h.CreateGuildCommand(h.Ready().Application.ID, discord.GuildID(guildID), command.DiscordData)
					if err != nil {
						log.Println("failed registering command", command.Name, guildID, err)
						continue
					}
					log.Println("registered command", name, "for", guildID)
				}
				continue
			}

			commands = append(commands, command.DiscordData)
			// h.AddFunc(command.Name, command.DiscordHandler)
			log.Println("registered command", name)
		}

		if err := cmdroute.OverwriteCommands(h.State, commands); err != nil {
			log.Println("cannot register global commands:", err)
		}
	})

	h.State.AddHandler(func(e *gateway.InteractionCreateEvent) {
		var respData *api.InteractionResponseData
		var ack bool

		go func() {
			switch d := e.Data.(type) {
			case *discord.CommandInteraction:
				if command := cmd.CommandMap[d.Name]; command.DiscordHandler != nil {
					cmdData := cmdroute.CommandData{
						Event: &e.InteractionEvent,
						Data:  d,
					}

					if len(d.Options) != 0 {
						cmdData.CommandInteractionOption = d.Options[0]
					}

					respData = command.DiscordHandler(h.Context(), cmdData)
					if respData == nil {
						return
					}
					break
				}
				respData = cmd.ErrorResponse(errors.New("unknown command"))
			default:
				log.Println("unhandled interaction event:", d)
				return
			}

			var err error

			if ack {
				_, err = h.State.FollowUpInteraction(e.AppID, e.Token, *respData)
			} else {
				ack = true
				err = h.State.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
					Type: api.MessageInteractionWithSource,
					Data: respData,
				})
			}

			if err != nil {
				log.Println("failed to send interaction callback:", err)
			}
		}()

		go func() {
			time.Sleep(time.Second * 2)
			if !ack {
				ack = true
				h.State.RespondInteraction(e.ID, e.Token, api.InteractionResponse{Type: api.DeferredMessageInteractionWithSource})
			}
		}()
	})

	h.State.AddHandler(func(e *gateway.MessageCreateEvent) {
		if e.Author.Bot {
			return
		}

		ctx := context.Background()
		var res client.TwitchBanStatusResponse

		cacheKey := e.GuildID.String() + ":" + e.Author.ID.String()

		cache := banStatusCache[cacheKey]
		if time.Now().After(cache.expiry) {
			guild, err := h.State.Guild(e.GuildID)
			if err != nil {
				log.Println("failed getting guild:", err)
				return
			}

			guildKey := fmt.Sprintf("%s:DTbinds", e.GuildID)
			twitchChannelID, err := client.RedisDB.HGet(ctx, guildKey, guild.OwnerID.String()).Result()
			if twitchChannelID == "" {
				if err != nil && err != redis.Nil {
					log.Println("failed getting Twitch channel id:", err)
				}
				return
			}

			twitchUserID, err := client.RedisDB.HGet(ctx, guildKey, e.Author.ID.String()).Result()
			if twitchUserID == "" {
				if err != nil && err != redis.Nil {
					log.Println("failed getting Twitch user id:", err)
				}
				return
			}

			res, err := client.GetTwitchBan(twitchChannelID, twitchUserID)
			if err != nil {
				log.Println("failed fetching ban status:", err)
				return
			}

			banStatusCache[cacheKey] = banCache{
				expiry:                  time.Now().Add(1 * time.Minute),
				TwitchBanStatusResponse: &res,
			}
		} else {
			res = *cache.TwitchBanStatusResponse
		}

		bannedUser := res.Data.BanStatus.BannedUser
		if bannedUser.ID != "" {
			// user is currently banned
			logReason := api.AuditLogReason(fmt.Sprintf("User banned on Twitch: %v", bannedUser))
			h.State.DeleteMessage(e.ChannelID, e.Message.ID, logReason)

			timeoutDuration := discord.NewTimestamp(time.Now().Add(3 * time.Minute))
			err := h.ModifyMember(e.GuildID, e.Author.ID, api.ModifyMemberData{
				CommunicationDisabledUntil: &timeoutDuration,
				AuditLogReason:             logReason,
			})
			if err != nil {
				log.Println("couldn't timeout member", err)
			}
		}
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := h.State.Connect(ctx); err != nil {
		log.Fatalln("cannot connect:", err)
	}
}
