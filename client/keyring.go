package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func setupKeyring(cdc codec.Codec) keyring.Keyring {
	k := keyring.NewInMemory(cdc)
	return k
}

func txConfig() (client.TxConfig, codec.Codec) {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	return authtx.NewTxConfig(cdc, authtx.DefaultSignModes), cdc
}

func (c *Client) setupFactory() {
	sdk.GetConfig().SetBech32PrefixForAccount(c.accountPrefix, c.accountPrefix+"pub")

	txConfig, cdc := txConfig()
	keybase := setupKeyring(cdc)
	path := hd.CreateHDPath(c.coinType, 0, 0).String()
	r, err := keybase.NewAccount("faucet-test", c.faucetMnemonics, "", path, hd.Secp256k1)
	if err != nil {
		log.Fatal(err)
	}
	pubkey, err := r.GetPubKey()
	if err != nil {
		log.Fatal(err)
	}
	accAddr := sdk.AccAddress(pubkey.Address())
	c.account = accAddr.String()
	factory := tx.Factory{}.WithKeybase(keybase).
		// WithGasPrices(c.gasPrices).
		WithChainID(c.chainID).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithTxConfig(txConfig).
		WithFromName("faucet-test")
	c.txFactory = factory
	c.txConfig = txConfig
}

func (c *Client) getAccountInfo(ctx context.Context, address string) (AccountInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/cosmos/auth/v1beta1/account_info/%s", c.apiEndpoint, address), nil)
	if err != nil {
		return AccountInfoResponse{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AccountInfoResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AccountInfoResponse{}, err
	}
	var accountInfo AccountInfoResponse
	err = json.Unmarshal(body, &accountInfo)
	if err != nil {
		return AccountInfoResponse{}, err
	}
	return accountInfo, nil
}

func (c *Client) transfer(ctx context.Context, factory tx.Factory, txConfig client.TxConfig, to string, amount string) (string, error) {
	accountInfo, err := c.getAccountInfo(ctx, c.account)
	if err != nil {
		return "", err
	}

	coins, err := sdk.ParseCoinsNormalized(amount)
	if err != nil {
		return "", err
	}
	toAddr, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		return "", err
	}
	msg := banktypes.NewMsgSend(sdk.MustAccAddressFromBech32(c.account), toAddr, coins)

	accountNumber, err := strconv.ParseUint(accountInfo.AccountInfo.AccountNumber, 10, 64)
	if err != nil {
		return "", err
	}
	sequence, err := strconv.ParseUint(accountInfo.AccountInfo.Sequence, 10, 64)
	if err != nil {
		return "", err
	}
	factory.SimulateAndExecute()

	fees, err := sdk.ParseCoinNormalized(c.gasPrices)
	if err != nil {
		return "", err
	}
	fees.Amount = fees.Amount.Mul(sdkmath.NewInt(c.gasAmount))
	factory = factory.WithGas(uint64(c.gasAmount)).WithAccountNumber(accountNumber).WithSequence(sequence).WithFees(fees.String())
	txb, err := factory.BuildUnsignedTx(msg)
	if err != nil {
		return "", err
	}
	err = tx.Sign(ctx, factory, "faucet-test", txb, false)
	if err != nil {
		return "", err
	}
	txBytes, err := txConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return "", err
	}

	// tx, err := txConfig.TxDecoder()(txBytes)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	txHash := fmt.Sprintf("%X", tmhash.Sum(txBytes))
	// jxJsonBytes, err := txConfig.TxJSONEncoder()(tx)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	node, err := client.NewClientFromNode(c.rpcEndpoint)
	if err != nil {
		return "", err
	}
	res, err := node.BroadcastTxAsync(context.Background(), txBytes)
	if err != nil {
		log := ""
		if res != nil && res.Log != "" {
			log = res.Log
		}
		return txHash, fmt.Errorf("failed to broadcast tx: %w, log: %s", err, log)
	}
	if res.Code != 0 {
		return txHash, fmt.Errorf("tx failed: %d, log: %s", res.Code, res.Log)
	}
	return txHash, nil
}

func (c *Client) FaucetAddress() string {
	return c.account
}
