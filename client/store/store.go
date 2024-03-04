package store

import (
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
)

type Key struct {
	Hash   string
	User   discord.User
	Expiry time.Time
}

var RegisterKeys = make(map[string]*Key)
