package cmd

import (
	"context"
	"fmt"
	"runtime"

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
				DiscordCodeBlock("md",
					fmt.Sprintf("# %s\n- TotalAlloc: %v MiB\n- NumGC: %v\n- NumGoroutine: %v",
						runtime.Version(), m.TotalAlloc/1024/1024, m.NumGC, runtime.NumGoroutine()))

			return Response(res)
		},
	})
}
