#!/bin/bash

git clone https://github.com/ping-pub/explorer
cd explorer
rm -rf chains/mainnet/* chains/testnet/*
cp ../mini.json chains/mainnet
yarn install
yarn dev