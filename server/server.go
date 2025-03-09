package server

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/public-awesome/faucet/client"
	"github.com/public-awesome/faucet/config"
)

type SendRequest struct {
	ID          string `json:"id"`
	GuildID     string `json:"guild_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	User        string `json:"user"`
	UserID      string `json:"user_id"`
	Amount      string `json:"amount"`
	Address     string `json:"address"`
}

type SendResponse struct {
	ID          string `json:"id"`
	GuildID     string `json:"guild_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	User        string `json:"user"`
	UserID      string `json:"user_id"`
	TxHash      string `json:"tx_hash"`
	Success     bool   `json:"success"`
	Error       string `json:"error"`
}

type Server struct {
	requests  chan *SendRequest
	responses chan *SendResponse
	client    *client.Client
	log       *slog.Logger
	config    *config.Config

	store *Store
}

func NewServer(log *slog.Logger) (*Server, error) {
	config, err := config.NewConfig()
	if err != nil {

		return nil, err
	}
	client := client.New(
		client.WithRPC(config.ClientConfig.RPCEndpoint),
		client.WithAPI(config.ClientConfig.APIEndpoint),
		client.WithAccountPrefix(config.ClientConfig.AccountPrefix),
		client.WithFaucetMnemonics(config.FaucetMnemonics),
		client.WithCoinType(config.ClientConfig.CoinType),
		client.WithChainID(config.ClientConfig.ChainID),
		client.WithGasAmount(config.ClientConfig.GasAmount),
		client.WithGasPrices(config.ClientConfig.GasPrices),
	)

	store, err := NewStore(path.Join(config.StorePath, "faucet.db"))
	if err != nil {
		return nil, err
	}
	return &Server{
		requests:  make(chan *SendRequest),
		responses: make(chan *SendResponse),
		client:    client,
		log:       log,
		config:    config,
		store:     store,
	}, nil
}

func (s *Server) ProcessRequests(ctx context.Context) {

	for {
		select {
		case req := <-s.requests:
			s.log.Info("processing request", "request_id", req.ID, "channel", req.ChannelName, "user", req.User, "user_id", req.UserID, "amount", req.Amount, "address", req.Address)

			txHash, err := s.client.BankSend(ctx, req.Address, req.Amount)
			success := true
			var errMsg string

			if err != nil {
				success = false
				errMsg = err.Error()
				s.log.Error("error sending request", "error", err)
			}

			s.responses <- &SendResponse{
				ID:          req.ID,
				GuildID:     req.GuildID,
				ChannelID:   req.ChannelID,
				ChannelName: req.ChannelName,
				User:        req.User,
				UserID:      req.UserID,
				TxHash:      txHash,
				Success:     success,
				Error:       errMsg,
			}
			<-time.After(5 * time.Second)
		case <-ctx.Done():
			s.log.Info("stopping request processor")
			return
		}
	}
}
func (s *Server) welcomeMessage(ds *discordgo.Session) {
	if s.config.DisableWelcomeMessage {
		return
	}
	for _, guild := range ds.State.Guilds {
		channels, err := ds.GuildChannels(guild.ID)
		if err != nil {
			s.log.Error("error fetching channels for guild", "error", err, "guild", guild)
			continue
		}
		for _, channel := range channels {
			for faucetChannel := range s.config.FaucetChannelCoins {
				if faucetChannel == channel.Name {
					m, err := ds.ChannelMessageSend(channel.ID, fmt.Sprintf("Welcome to the Stargaze Faucet! Please use the `$request %s1zxcvaqswdedefr...` command to request tokens.", s.config.ClientConfig.AccountPrefix))
					if err != nil {
						s.log.Error("error sending welcome message", "error", err, "channel", channel)
						continue
					}
					if m != nil {
						s.log.Info("sent welcome message", "channel", channel, "message", m.ID)
					}
				}
			}
		}
	}
}
func (s *Server) Run(ctx context.Context) error {
	s.log.Info("starting server")
	s.log.Info("using faucet address", "address", s.client.FaucetAddress())

	defer func() {
		err := s.store.Close()
		if err != nil {
			s.log.Error("error closing store", "error", err)
		}
	}()

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + s.config.FaucetBotToken)
	if err != nil {
		s.log.Error("error creating discord session", "error", err)
		return err
	}

	dg.AddHandler(s.messageHandler)
	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		s.log.Error("error opening discord session", "error", err)
		return err
	}
	defer dg.Close()
	s.welcomeMessage(dg)
	go s.ProcessRequests(ctx)
	go s.processResponses(ctx, dg)

	<-ctx.Done()
	s.log.Info("stopping server")

	return nil
}
