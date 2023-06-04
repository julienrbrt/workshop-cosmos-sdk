#!/bin/bash

alice=$(minid keys show alice --address)
bob=$(minid keys show bob --address)
carol=$(minid keys show carol --address)

get_name () {
    for people in [$alice,$bob,$carol]; do
        stringify_name $1
    done;
}

stringify_name () {
    case $1 in 
        $alice)
            echo "alice"
            ;;
        $bob)
            echo "bob"
            ;;
        $carol)
            echo "carol"
            ;;
    esac
}

# Normally mempool were FIFO, so it the block should have been by time of transaction receival
# In this case CAROL - ALICE - BOB (try this with `minid start --mempool-type none`)
# However, with the fee mempool, the transactions or ordered by fees, so in the block it will be ordered as
# BOB - ALICE - CAROL (try this with `minid start --mempool-type fee``)
minid tx bank send carol $alice 10mini -y --output json > /dev/null
tx=$(minid tx bank send alice $bob 10mini --fees 10mini -y --output json | jq -r .txhash)
minid tx bank send bob $carol 10mini --fees 100mini -y --output json > /dev/null

echo "--> sleeping the block time timeout duration"
sleep 15s

# query which block those txs have been included into
height=$(minid q tx $tx --type hash --output json | jq .height -r)

# get all txs
txs=$(minid q block $height | jq .block.data.txs -r | jq -c '.[]')

echo "--> printing transaction order in block $height"
for rawTx in $txs; do
    get_name $(minid tx decode $(echo $rawTx | jq -r) | jq -r .body.messages[0].from_address)
done;