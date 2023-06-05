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
// This mempool is not optimized, do not use in production.
type FeeMempool struct {
	logger log.Logger
	pool   fmTxs
}

type fmTx struct {
	address  string
	priority int64
	tx       sdk.Tx
}

func (fm fmTx) Equal(other fmTx) bool {
	if fm.address != other.address || fm.priority != other.priority {
		return false
	}

	if len(fm.tx.GetMsgs()) != len(other.tx.GetMsgs()) {
		return false
	}

	for i, msg := range fm.tx.GetMsgs() {
		if msg.String() != other.tx.GetMsgs()[i].String() { // bad comparison but this is just for demonstration purpose
			return false
		}
	}

	return true
}

var _ mempool.Iterator = &fmTxs{}

type fmTxs struct {
	idx int
	txs []fmTx
}

// Next returns an interator with one less tx in the pool
func (fm *fmTxs) Next() mempool.Iterator {
	if len(fm.txs) == 0 {
		return nil
	}

	if len(fm.txs) == fm.idx+1 {
		return nil
	}

	fm.idx++
	return fm
}

func (fm *fmTxs) Tx() sdk.Tx {
	if fm.idx >= len(fm.txs) {
		panic(fmt.Sprintf("index out of bound: %d, fmTxs: %v", fm.idx, fm))
	}

	return fm.txs[fm.idx].tx
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
	// note, we could have got the prority from the context as well
	// however, we wanted to demonstrate that any custom logic can be used to determine the priority of a transaction
	// sdkContext := sdk.UnwrapSDKContext(ctx)
	// priority := sdkContext.Priority()
	var priority int64
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		priority = naiveGetTxPriority(feeTx.GetFee())
	}

	fm.logger.Info(fmt.Sprintf("transaction from %s inserted in mempool with priority %d", sender, priority))
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
	if len(fm.pool.txs) == 0 {
		return nil
	}

	// sort all txs after each insertion (truly not efficient, but you get it)
	sort.Slice(fm.pool.txs, func(i, j int) bool {
		return fm.pool.txs[j].priority < fm.pool.txs[i].priority
	})

	return &fm.pool
}

// CountTx returns the total amount of transactions in the mempool
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
		priority = naiveGetTxPriority(feeTx.GetFee())
	}

	txToDelete := fmTx{priority: priority, address: sender, tx: tx}
	for idx, fmTx := range fm.pool.txs {
		if fmTx.Equal(txToDelete) {
			fm.pool.txs = removeAtIndex(fm.pool.txs, idx)
			return nil
		}
	}

	return mempool.ErrTxNotFound
}

// naiveGetTxPriority returns a naive tx priority based on the amount of the smallest denomination of the fee
// provided in a transaction.
func naiveGetTxPriority(fee sdk.Coins) int64 {
	var priority int64
	for _, c := range fee {
		p := int64(math.MaxInt64)
		if c.Amount.IsInt64() {
			p = c.Amount.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority
}
