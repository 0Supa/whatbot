package client

import (
	"github.com/redis/go-redis/v9"
)

var RedisDB = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       9,
})
