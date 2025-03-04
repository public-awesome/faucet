package config_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/public-awesome/faucet/config"
	env "github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	os.Setenv("FAUCET_CHANNEL_AMOUNTS", "faucet:10_000_000ustars;private-faucet:10_000_000ustars,1factory/stars123456789;🚰│faucet:1ustars,1uatom,1uinit;1234567891012345:1ustars")
	os.Setenv("FAUCET_CHANNEL_INTERVAL", "faucet:1h;private-faucet:190h")
	cfg := &config.Config{}
	err := env.Process(context.Background(), cfg)
	assert.NoError(t, err)

	assert.Equal(t, cfg.FaucetChannelCoins, map[string]config.ChannelConfig{
		"faucet":           {Coins: "10000000ustars"},
		"private-faucet":   {Coins: "10000000ustars,1factory/stars123456789"},
		"🚰│faucet":         {Coins: "1ustars,1uatom,1uinit"},
		"1234567891012345": {Coins: "1ustars"},
	})

	assert.Equal(t, cfg.FaucetChannelInterval, map[string]time.Duration{
		"faucet":         1 * time.Hour,
		"private-faucet": 190 * time.Hour,
	})
}
