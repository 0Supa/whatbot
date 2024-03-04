package client

import (
	"os"

	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/state"
	_ "github.com/joho/godotenv/autoload"
)

type handler struct {
	*cmdroute.Router
	*state.State
}

func newHandler(s *state.State) *handler {
	h := &handler{State: s}

	h.Router = cmdroute.NewRouter()

	return h
}

var Handler = newHandler(state.New("Bot " + os.Getenv("DISCORD_TOKEN")))
