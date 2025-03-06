package config

import (
	"context"
	"strings"
	"time"

	env "github.com/sethvargo/go-envconfig"
)

type Config struct {
	FaucetMnemonics string `env:"FAUCET_MNEMONICS"`
	// FaucetChannelAmounts is a map of channel name to amount of tokens to send
	// Example: FAUCET_CHANNEL_AMOUNTS="faucet:10_000_000ustars;private-faucet:10_000_000ustars"
	FaucetChannelCoins map[string]ChannelConfig `env:"FAUCET_CHANNEL_AMOUNTS, delimiter=;,separator=:"`
	// FaucetChannelInterval is a map of channel name to interval of time to send tokens
	// Example: FAUCET_CHANNEL_INTERVAL="faucet:1h;private-faucet:190h"
	FaucetChannelInterval map[string]time.Duration `env:"FAUCET_CHANNEL_INTERVAL, delimiter=;,separator=:"`

	FaucetBotToken string `env:"FAUCET_BOT_TOKEN, required"`

	ClientConfig ClientConfig `env:",prefix=FAUCET_CLIENT_"`
	ExplorerURL  string       `env:"FAUCET_EXPLORER_URL"`

	StorePath string `env:"FAUCET_STORE_PATH, default=faucet-data"`

	DisableWelcomeMessage bool `env:"DISABLE_WELCOME_MESSAGE, default=false"`
}

type ClientConfig struct {
	RPCEndpoint   string `env:"RPC_ENDPOINT, required"`
	APIEndpoint   string `env:"API_ENDPOINT, required"`
	AccountPrefix string `env:"ACCOUNT_PREFIX, required"`
	GasAmount     int64  `env:"GAS_AMOUNT, default=500000"`
	GasPrices     string `env:"GAS_PRICES, required"`
	CoinType      uint32 `env:"COIN_TYPE, default=118"`
	ChainID       string `env:"CHAIN_ID, required"`
}

type ChannelConfig struct {
	Coins string `json:"coins"`
}

func (cg *ChannelConfig) UnmarshalText(text []byte) error {
	cg.Coins = string(text)
	coinsRaw := strings.Split(cg.Coins, ",")
	coins := make([]string, 0, len(coinsRaw))
	for _, coin := range coinsRaw {
		coins = append(coins, strings.ReplaceAll(strings.TrimSpace(coin), "_", ""))
	}
	cg.Coins = strings.Join(coins, ",")
	return nil
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := env.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
