package mempool

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/cometbft/cometbft/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var _ mempool.Mempool = (*FeeMempool)(nil)

func NewFeeMempool(logger log.Logger) *FeeMempool {
	return &FeeMempool{logger: logger.With("module", "fee-mempool")}
}

// FeeMempool defines a mempool that prioritizes transactions according to their fees.
// Transactions with higher fees are placed at the front of the queue.
// Once no more transactions has fees, the remainaing transactions are inserted until the mempool is full.
type FeeMempool struct {
	logger log.Logger
	pool   fmTxs
}

type fmTx struct {
	address  string
	priority int64
	tx       sdk.Tx
}

var _ mempool.Iterator = &fmTxs{}

type fmTxs struct {
	lastTx bool
	txs    []fmTx
}

// Next returns an interator with one less tx in the pool
func (fm *fmTxs) Next() mempool.Iterator {
	if len(fm.txs) == 0 || fm.lastTx {
		return nil
	} else if len(fm.txs) == 1 {
		return &fmTxs{txs: fm.txs, lastTx: true}
	}

	fm.txs = removeAtIndex(fm.txs, 0)
	return fm
}

func (fm *fmTxs) Tx() sdk.Tx {
	return fm.txs[0].tx
}

// Insert a transaction in the mempool per sender
func (fm *FeeMempool) Insert(_ context.Context, tx sdk.Tx) error {
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

	fm.logger.Info(fmt.Sprintf("transaction from %s inserted in mempool", sender))
	fm.pool.txs = append(fm.pool.txs, fmTx{
		address:  sender,
		priority: priority,
		tx:       tx,
	})

	return nil
}

// Select returns an iterator ordering transactions the mempool with the highest fee.
// NOTE: It is not safe to use this iterator while removing transactions from the underlying mempool.
func (fm *FeeMempool) Select(_ context.Context, _ [][]byte) mempool.Iterator {
	// sort all txs after each insertion (truly not efficient, but you get it)
	sort.Slice(fm.pool.txs, func(i, j int) bool {
		return fm.pool.txs[i].priority < fm.pool.txs[j].priority
	})

	return &fm.pool
}

// CounTx counts the amount of txs in the mempool
func (fm *FeeMempool) CountTx() int {
	return len(fm.pool.txs)
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

	txToDelete := fmTx{priority: priority, address: sender, tx: tx}
	for idx, fmTx := range fm.pool.txs {
		if fmTx == txToDelete {
			fm.pool.txs = removeAtIndex(fm.pool.txs, idx)
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
