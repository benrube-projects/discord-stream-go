package discord

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type RWMap struct {
	sync.RWMutex
	Data map[string]bool
}

type DiscordClient struct {
	Logger             *zap.Logger
	client             *discordgo.Session
	guildsTransmitting RWMap
}

var buffer = make([][]byte, 0)

func init() {
	loadSound()
}

func (c *DiscordClient) StartServer() {
	c.guildsTransmitting = RWMap{
		Data: make(map[string]bool),
	}
	c.Logger.Info("Starting Discord server")
	var err error
	c.client, err = discordgo.New(TokenPrefix + os.Getenv(TokenConfig))
	if err != nil {
		c.Logger.Fatal("error creating Discord session", zap.Error(err))
	}

	c.register()
	err = c.client.Open()
	if err != nil {
		c.Logger.Fatal("error opening connection", zap.Error(err))
	}

	c.Logger.Info("Discord server started")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	c.client.Close()
	c.Logger.Info("Discord server stopped")

}

// Register discord events
func (c *DiscordClient) register() {
	c.client.AddHandler(c.ready)
	c.client.AddHandler(c.handleMessages())
	c.client.AddHandler(c.handleVoiceStateUpdate)
}

func (c *DiscordClient) handleMessages() func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}

		c.Logger.Info("got message", zap.String("msg", m.Message.Content))
	}
}

// Discord Handler for detecting when someone joins a voice channel
func (c *DiscordClient) handleVoiceStateUpdate(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	if m.UserID != s.State.User.ID {
		// If there is no before state then the user has joined a channel
		if m.BeforeUpdate == nil {
			c.client.RWMutex.RLock()
			if _, found := c.guildsTransmitting.Data[m.GuildID]; found {
				c.client.RWMutex.RUnlock()
				c.Logger.Warn("already transmitting", zap.String("guild", m.GuildID))
				return
			} else {
				c.client.RWMutex.RUnlock()
				c.guildsTransmitting.Lock()
				c.guildsTransmitting.Data[m.GuildID] = true
				c.guildsTransmitting.Unlock()
				defer func() {
					c.guildsTransmitting.Lock()
					delete(c.guildsTransmitting.Data, m.GuildID)
					c.guildsTransmitting.Unlock()
				}()
			}
			vc, err := s.ChannelVoiceJoin(m.GuildID, m.ChannelID, false, true)
			if err != nil {
				c.Logger.Error("error joining voice channel", zap.Error(err))
			}
			c.Logger.Info("Joined voice channel", zap.String("channel", m.ChannelID))
			time.Sleep(250 * time.Millisecond)
			vc.Speaking(true)
			for _, buff := range buffer {
				vc.OpusSend <- buff
			}

			vc.Speaking(false)
			time.Sleep(250 * time.Millisecond)
			// Disconnect from the provided voice channel.
			vc.Disconnect()

		} else {
			c.Logger.Info("Left voice channel", zap.String("channel", m.ChannelID))
		}
	}
}
func (c *DiscordClient) ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	c.client.UpdateGameStatus(0, "yummy")
}

func loadSound() error {

	file, err := os.Open("/Users/ben/Downloads/test.dca")
	if err != nil {
		fmt.Println("Error opening dca file :", err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			fmt.Println("Error reading from dca file :", err)
			return err
		}

		// Append encoded pcm data to the buffer.
		buffer = append(buffer, InBuf)
	}
}
