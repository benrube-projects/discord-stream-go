package main

import (
	"github.com/ben-rube/discord-stream-go/internal/discord"
	"go.uber.org/zap"
)

func main() {
	// New Zap logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	client := discord.DiscordClient{

		Logger: logger,
	}

	client.StartServer()

}
