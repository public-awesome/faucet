package client

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Client struct {
	rpcEndpoint     string
	apiEndpoint     string
	accountPrefix   string
	faucetMnemonics string
	coinType        uint32
	chainID         string
	gasPrices       string
	gasAmount       int64
	account         string

	txConfig  client.TxConfig
	txFactory tx.Factory
}

type ClientOption func(*Client)

func WithRPC(rpc string) ClientOption {
	return func(c *Client) {
		c.rpcEndpoint = rpc
	}
}

func WithAPI(api string) ClientOption {
	return func(c *Client) {
		c.apiEndpoint = api
	}
}

func WithAccountPrefix(prefix string) ClientOption {
	return func(c *Client) {
		c.accountPrefix = prefix
	}
}

func WithFaucetMnemonics(mnemonics string) ClientOption {
	return func(c *Client) {
		c.faucetMnemonics = mnemonics
	}
}

func WithCoinType(coinType uint32) ClientOption {
	return func(c *Client) {
		c.coinType = coinType
	}
}

func WithChainID(chainID string) ClientOption {
	return func(c *Client) {
		c.chainID = chainID
	}
}

func WithGasPrices(gasPrices string) ClientOption {
	return func(c *Client) {
		c.gasPrices = gasPrices
	}
}

func WithGasAmount(gasAmount int64) ClientOption {
	return func(c *Client) {
		c.gasAmount = gasAmount
	}
}

func New(opts ...ClientOption) *Client {
	c := &Client{coinType: 118}
	for _, opt := range opts {
		opt(c)
	}
	c.setupFactory()
	return c
}

func (c *Client) BankSend(ctx context.Context, address, amount string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.transfer(timeoutCtx, c.txFactory, c.txConfig, address, amount)
}

func (c *Client) ValidAddress(address string) bool {
	_, err := sdk.GetFromBech32(address, c.accountPrefix)
	return err == nil
}
