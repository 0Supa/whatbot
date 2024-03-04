package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	discordClient "github.com/0supa/degen-go/client/discord"

	"github.com/0supa/degen-go/client"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

func systemPrompt(event discord.InteractionEvent) string {
	bot := discordClient.Handler.Ready()
	return fmt.Sprintf(`You are '%s', a Discord bot running a large language model.

Current time: %s

You can use Markdown syntax for formatting your response, excluding tables and images.
You can mention or ping channels and users using the following format: '<#channel_id>', respectively '<@user_id>'. 

Do NOT refer to commands, since you don't know any.
Do NOT add opening or closing sentences.

Context:
- Channel name: %s
- Channel ID: %s
- Channel NSFW flag: %t
- User ID: %s
- User name: %s

Prompt:`,
		bot.User.DisplayName,
		time.Now().Format("2006-01-02 15:04:05"),
		event.Channel.Name, event.Channel.ID, event.Channel.NSFW, event.Member.User.ID, event.Member.User.DisplayName)
}

func init() {
	var model = "@cf/mistral/mistral-7b-instruct-v0.1"
	RegisterCommand(Command{
		Name: "ask",
		DiscordData: api.CreateCommandData{
			Name:        "ask",
			Description: "Mistral 7B Text Generation LLM",
			Options: []discord.CommandOption{
				&discord.StringOption{
					OptionName:  "prompt",
					Description: "AI Prompt",
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

			query := client.TextQuery{
				Stream: true,
				Messages: []client.QueryMessage{
					{
						Role:    "system",
						Content: systemPrompt(*cmd.Event),
					},
					{
						Role:    "user",
						Content: options.Prompt,
					},
				},
			}

			c := make(chan client.Result)
			go client.TextGeneration(c, query, model)

			var builder strings.Builder

			var replied bool
			updateMsg := func() {
				if !replied {
					replied = true
					res := api.InteractionResponse{
						Type: api.MessageInteractionWithSource,
						Data: Response(builder.String()),
					}

					discordClient.Handler.State.RespondInteraction(cmd.Event.ID, cmd.Event.Token, res)
					return
				}

				discordClient.Handler.State.EditInteractionResponse(cmd.Event.AppID, cmd.Event.Token, api.EditInteractionResponseData{
					Content: option.NewNullableString(builder.String()),
				})
			}

			var lastUpdate time.Time
			for data := range c {
				if err := data.Error; err != nil {
					return ErrorResponse(err)
				}

				builder.WriteString(data.Response)
				if time.Since(lastUpdate) > 1*time.Second && strings.TrimSpace(data.Response) != "" {
					go updateMsg()
					lastUpdate = time.Now()
				}
			}
			updateMsg()

			return nil
			// return Response(builder.String())
		},
	})
}
