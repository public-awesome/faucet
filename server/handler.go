package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid"
)

func parts(message, accountPrefix string) []string {
	message = strings.TrimSpace(message)
	parts := strings.Split(message, " ")
	filteredPars := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filteredPars = append(filteredPars, part)
	}
	if len(filteredPars) < 2 || filteredPars[0] != "$request" || !strings.HasPrefix(filteredPars[1], accountPrefix) {
		return nil
	}
	return filteredPars
}

func (s *Server) block(channelId, address, author string, waitPeriod time.Duration) (bool, time.Duration) {
	s.trackMu.Lock()
	defer s.trackMu.Unlock()

	addressKey := fmt.Sprintf("%s-%s", channelId, address)
	authorKey := fmt.Sprintf("%s-%s", channelId, author)
	previous, ok := s.track[addressKey]
	if ok && time.Since(previous) < waitPeriod {
		return true, waitPeriod - time.Since(previous)
	}
	previousAuthor, ok := s.trackByAuthor[authorKey]
	if ok && time.Since(previousAuthor) < waitPeriod {
		return true, waitPeriod - time.Since(previousAuthor)
	}

	s.track[addressKey] = time.Now()
	s.trackByAuthor[authorKey] = time.Now()
	return false, 0
}
func (s *Server) messageHandler(ds *discordgo.Session, message *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if message.Author.ID == ds.State.User.ID {
		return
	}

	// Fetch channel details
	channel, err := ds.Channel(message.ChannelID)
	if err != nil {
		s.log.Error("error fetching channel details", "error", err, "channel", message.ChannelID)
		return
	}

	_, ok := s.config.FaucetChannelCoins[channel.Name]
	// not configured for this channel
	if !ok {
		return
	}

	faucetInterval, ok := s.config.FaucetChannelInterval[channel.Name]
	if !ok {
		faucetInterval = time.Hour * 24 * 5
	}
	id := fmt.Sprintf("%s-%s", channel.GuildID, channel.ID)
	block, waitTime := s.block(id, message.Author.Username, message.Author.ID, faucetInterval)
	if block {
		reply := fmt.Sprintf("<@%s> you can send a request again <t:%d:R>", message.Author.ID, time.Now().Add(waitTime).UTC().Unix())
		_, err := ds.ChannelMessageSend(message.ChannelID, reply)
		if err != nil {
			log.Printf("Error sending message: %v", err)
		}
		return
	}
	requestID, err := uuid.NewV7()
	if err != nil {
		s.log.Error("error generating uuid", "error", err)
		return
	}
	parts := parts(message.Content, s.config.ClientConfig.AccountPrefix)

	if len(parts) == 2 {
		reply := fmt.Sprintf("<@%s> invalid request, please use the `$request <address>` command", message.Author.ID)
		_, err = ds.ChannelMessageSend(message.ChannelID, reply)
		if err != nil {
			s.log.Error("error sending message", "error", err)
		}
		return
	}
	req := &SendRequest{
		ID:          requestID.String(),
		GuildID:     channel.GuildID,
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		User:        message.Author.Username,
		UserID:      message.Author.ID,
		Amount:      s.config.FaucetChannelCoins[channel.Name].Coins,
		Address:     parts[1],
	}
	s.log.Info("sending request", "request_id", req.ID, "channel", req.ChannelName, "user", req.User, "user_id", req.UserID, "amount", req.Amount, "address", req.Address)

	s.requests <- req
	reply := fmt.Sprintf("<@%s> your request has been sent, the transaction will be broadcasted in a few seconds", message.Author.ID)
	_, err = ds.ChannelMessageSend(message.ChannelID, reply)
	if err != nil {
		s.log.Error("error sending message", "error", err)
	}

}

func (s *Server) processResponses(ctx context.Context, ds *discordgo.Session) {
	for {
		select {
		case response := <-s.responses:
			s.log.Info("processing response", "response_id", response.ID, "channel", response.ChannelName, "user", response.User, "user_id", response.UserID, "tx_hash", response.TxHash, "success", response.Success, "error", response.Error)
			if response.Success {
				reply := fmt.Sprintf("<@%s> your request has been sent, check your transaction %s/%s", response.UserID, s.config.ExplorerURL, response.TxHash)
				_, err := ds.ChannelMessageSend(response.ChannelID, reply)
				if err != nil {
					s.log.Error("error sending message", "error", err)
				}
			} else {
				reply := fmt.Sprintf("<@%s> your request has failed, please try again later", response.UserID)
				_, err := ds.ChannelMessageSend(response.ChannelID, reply)
				if err != nil {
					s.log.Error("error sending message", "error", err)
				}
			}
		case <-ctx.Done():
			s.log.Info("stopping response processor")
			return
		}
	}

}
