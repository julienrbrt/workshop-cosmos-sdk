package mempool

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ mempool.Mempool = (*FeeMempool)(nil)

func NewFeeMempool() FeeMempool {
	return FeeMempool{}
}

// FeeMempool defines a mempool that prioritizes transactions according to their fees.
// Transactions with higher fees are placed at the front of the queue.
type FeeMempool struct{}

func (FeeMempool) Insert(context.Context, sdk.Tx) error {
	return nil
}

func (FeeMempool) Select(context.Context, [][]byte) mempool.Iterator {
	return nil
}

func (FeeMempool) CountTx() int {
	return 0
}

func (FeeMempool) Remove(sdk.Tx) error {
	return nil
}
