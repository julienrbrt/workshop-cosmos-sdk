#!/bin/bash

rm -r ~/.minid || true
MINID_BIN=$(which minid)
CONFIG_FOLDER=~/.minid/config
# configure minid
$MINID_BIN config chain-id demo
$MINID_BIN config keyring-backend test
$MINID_BIN keys add alice
$MINID_BIN keys add bob
$MINID_BIN keys add carol
$MINID_BIN init test --chain-id demo --default-denom mini
# update genesis
$MINID_BIN genesis add-genesis-account alice 10000000mini --keyring-backend test
$MINID_BIN genesis add-genesis-account bob 1000mini --keyring-backend test
$MINID_BIN genesis add-genesis-account carol 1000mini --keyring-backend test
# create default validator
$MINID_BIN genesis gentx alice 1000000mini --chain-id demo
$MINID_BIN genesis collect-gentxs
# edit app.toml (enable api, swagger and unsafe cors)
go install github.com/tomwright/dasel/cmd/dasel@272b38fee3a2 # this will not be necessary after Cosmos SDK v0.50 and Confix
dasel put bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.enable'
dasel put bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.swagger'
dasel put bool -f $CONFIG_FOLDER/app.toml -v "true" '.api.enabled-unsafe-cors'