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

func timetoBytes(t time.Time) ([]byte, error) {
	return t.MarshalBinary()
}

func bytesToTime(b []byte) (time.Time, error) {
	var t time.Time
	err := t.UnmarshalBinary(b)
	return t, err
}
func (s *Server) getByKey(key string) (*time.Time, error) {
	lastRequest, err := s.store.Get([]byte(key))
	if err != nil && err != ErrNotFound {
		return nil, err
	}
	if err == ErrNotFound {
		return nil, nil
	}
	lastRequestTime, err := bytesToTime(lastRequest)
	if err != nil {
		return nil, err
	}
	return &lastRequestTime, nil

}

func (s *Server) block(channelId, address, author string, waitPeriod time.Duration) (bool, time.Duration) {

	// track by recipient and by discord user id
	addressKey := fmt.Sprintf("%s-%s", channelId, address)
	authorKey := fmt.Sprintf("%s-%s", channelId, author)

	lastRequestByAddress, err := s.getByKey(addressKey)
	if err != nil && err != ErrNotFound {
		s.log.Error("error getting address key", "error", err, "address_key", addressKey)
	}

	if lastRequestByAddress != nil && time.Since(*lastRequestByAddress) < waitPeriod {
		waitPeriod = waitPeriod - time.Since(*lastRequestByAddress)
		return true, waitPeriod
	}

	lastRequestByAuthor, err := s.getByKey(authorKey)
	if err != nil && err != ErrNotFound {
		s.log.Error("error getting author key", "error", err, "author_key", authorKey)
	}
	if lastRequestByAuthor != nil && time.Since(*lastRequestByAuthor) < waitPeriod {
		waitPeriod = waitPeriod - time.Since(*lastRequestByAuthor)
		return true, waitPeriod
	}
	now, err := timetoBytes(time.Now())
	if err != nil {
		s.log.Error("error getting now", "error", err)
		return false, 0
	}
	s.store.Set([]byte(addressKey), now)
	s.store.Set([]byte(authorKey), now)
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

	parts := parts(message.Content, s.config.ClientConfig.AccountPrefix)
	if len(parts) == 2 && parts[0] != "$request" {
		reply := fmt.Sprintf("<@%s> invalid request, please use the `$request <address>` command", message.Author.ID)
		_, err = ds.ChannelMessageSend(message.ChannelID, reply)
		if err != nil {
			s.log.Error("error sending message", "error", err)
		}
		return
	}
	if len(parts) != 2 {
		s.log.Info("invalid request", "request", message.Content)
		return
	}
	valid := s.client.ValidAddress(parts[1])
	if !valid {
		reply := fmt.Sprintf("<@%s> invalid address, please use a valid address", message.Author.ID)
		_, err = ds.ChannelMessageSend(message.ChannelID, reply)
		if err != nil {
			s.log.Error("error sending message", "error", err)
		}
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
