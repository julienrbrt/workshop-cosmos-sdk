package mempool

import (
	"context"
	"fmt"
	"math"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var _ mempool.Mempool = (*FeeMempool)(nil)

func NewFeeMempool(maxTx int) *FeeMempool {
	return &FeeMempool{}
}

// FeeMempool defines a mempool that prioritizes transactions according to their fees.
// Transactions with higher fees are placed at the front of the queue.
// Once no more transactions has fees, the remainaing transactions are inserted until the mempool is full.
type FeeMempool struct {
	txs   fmTxs
	maxTx int
}

type fmTx struct {
	address  string
	priority int64
	tx       sdk.Tx
}

var _ mempool.Iterator = fmTxs{}

type fmTxs []fmTx

// Next returns an interator with one less tx in the pool
func (fm fmTxs) Next() mempool.Iterator {
	if len(fm) == 0 {
		return nil
	}

	return fm[1 : len(fm)-1]
}

func (fm fmTxs) Tx() sdk.Tx {
	if len(fm) == 0 {
		return nil
	}

	return fm[0].tx
}

// Insert a transaction in the mempool per sender
func (fm *FeeMempool) Insert(_ context.Context, tx sdk.Tx) error {
	// check the mempool capacity if a maximum amount of transaction is set
	if fm.maxTx > 0 && fm.CountTx() >= fm.maxTx {
		return mempool.ErrMempoolTxMaxCapacity
	}
	if fm.maxTx < 0 {
		return nil
	}

	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("tx must have at least one signer")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()

	// by default a transaction has no priority
	var priority int64
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		priority = naiveGetTxPriority(feeTx.GetFee(), int64(feeTx.GetGas()))
	}

	fm.txs = append(fm.txs, fmTx{
		priority: priority,
		address:  sender,
		tx:       tx,
	})

	return nil
}

// Select returns an iterator ordering transactions the mempool with the highest fee.
// NOTE: It is not safe to use this iterator while removing transactions from the underlying mempool.
func (fm *FeeMempool) Select(_ context.Context, _ [][]byte) mempool.Iterator {
	// sort all txs after each insertion (truly not efficient, but you get it)
	sort.Slice(fm.txs, func(i, j int) bool {
		return fm.txs[i].priority > fm.txs[j].priority
	})

	return fm.txs
}

// CounTx counts the amount of txs in the mempool
func (fm *FeeMempool) CountTx() int {
	return len(fm.txs)
}

// Remove removes a tx from the mempool. It returns an error if the tx does not have at least one signer or the tx was not found in the pool.
func (fm *FeeMempool) Remove(tx sdk.Tx) error {
	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("tx must have at least one signer")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	var priority int64
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		priority = naiveGetTxPriority(feeTx.GetFee(), int64(feeTx.GetGas()))
	}

	txToDelete := fmTx{priority: priority, address: sender}
	for idx, fmTx := range fm.txs {
		if fmTx == txToDelete {
			fm.txs = removeAtIndex(fm.txs, idx)
			return nil
		}
	}

	return fmt.Errorf("failed to remove transaction from the mempool")
}

// took from https://github.com/cosmos/cosmos-sdk/blob/9f9833e518df0c3ce3816a3eb369666dedacf4c3/x/auth/ante/validator_tx_fee.go#L50 for demonstration purpose
// naiveGetTxPriority returns a naive tx priority based on the amount of the smallest denomination of the gas price
// provided in a transaction.
// NOTE: This implementation should be used with a great consideration as it opens potential attack vectors
// where txs with multiple coins could not be prioritize as expected.
func naiveGetTxPriority(fee sdk.Coins, gas int64) int64 {
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		gasPrice := c.Amount.QuoRaw(gas)
		if gasPrice.IsInt64() {
			p = gasPrice.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}
