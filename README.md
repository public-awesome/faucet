# Faucet

A simple faucet for Cosmos SDK chains.

## Build

```bash
make build
```

## Configuration

The faucet is configured using environment variables, the following variables are supported:

The `FAUCET_CHANNEL_AMOUNTS` variable list of channel names or channel ids and the amount of tokens to send to each channel it suppors multiple coins separated by commas and multiple channels separated by semicolons. It also supports underscores for integer literals to make them easier to read.

The `FAUCET_CHANNEL_INTERVAL` variable is a comma-separated list of channel names and the interval of time to wait before allowing another request by the same user or recipient address. If no interval is provided for a channel the default of 1 hour will be used.

The `FAUCET_CLIENT_CHAIN_ID` variable is the id of the chain the faucet is running on.

The `FAUCET_CLIENT_GAS_PRICES` variable is the gas price to use for the faucet transactions.

The `FAUCET_CLIENT_RPC_URL` variable is the url of the rpc endpoint of the chain the faucet is running on.

The `FAUCET_CLIENT_API_ENDPOINT` variable is the url of the rest/api endpoint of the chain the faucet is running on.

The `FAUCET_BOT_TOKEN` variable is the token of the discord bot that will be used to send messages to the users and to listen for requests.

## Usage with binary

```bash
export FAUCET_CHANNEL_AMOUNTS="faucet:10_000_000ustars;private-faucet:10_000_000ustars,1factory/stars123456789;🚰│faucet:1ustars,1uatom,1uinit;1234567891012345:1ustars"
export FAUCET_CHANNEL_INTERVAL="faucet:1h;private-faucet:190h"
export FAUCET_MNEMONICS="some mnemonic for the faucet account either 12 or 24 words can be used"
export PORT=8080
export FAUCET_CHAIN_ID="elgafar-1"
export FAUCET_GAS_PRICES="1ustars"
export FAUCET_GAS_ADJUSTMENT=1.7
export FAUCET_RPC_URL="https://rpc.elgafar-1.stargaze-apis.com:443"
export FAUCET_API_ENDPOINT="https://rest.elgafar-1.stargaze-apis.com"
export FAUCET_BOT_TOKEN="your-discord-bot-token"
./bin/faucet-server
```

## Usage with docker

Modify the `faucet.env` file with the correct values and then run the following command:

```bash
docker compose up -d
```
