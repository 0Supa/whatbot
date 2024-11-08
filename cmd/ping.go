package cmd

import (
	"context"
	"fmt"
	"runtime"

	discordClient "github.com/0supa/whatbot/client/discord"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
)

func init() {
	RegisterCommand(Command{
		Name: "ping",
		DiscordData: api.CreateCommandData{
			Name:        "ping",
			Description: "Ping",
		},
		DiscordHandler: func(ctx context.Context, cmd cmdroute.CommandData) *api.InteractionResponseData {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			res := "# <a:gopherDance:1203488020389437480> Pong!\n" +
				CodeBlock("md",
					fmt.Sprintf(
						`
# Discord
- Gateway: %v ms
- Guilds: %v
# %s
- Alloc: %v MiB
- NumGC: %v
- NumGoroutine: %v`,
						// Discord
						discordClient.Handler.Gateway().Latency().Milliseconds(),
						len(discordClient.Handler.Ready().Guilds),
						// go
						runtime.Version(),
						m.Alloc/1024/1024,
						m.NumGC,
						runtime.NumGoroutine(),
					),
				)

			return Response(res)
		},
	})
}
