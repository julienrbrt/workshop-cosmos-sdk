## Creating a Fee Mempool

### Step 1: Import Necessary Packages

At the top of the file, you will need to import necessary packages:

```go
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

```

### Step 2: Define FeeMempool

Next, we'll define our `FeeMempool` struct.

```go
type FeeMempool struct {
	txs   fmTxs
	maxTx int
}
```

This struct will contain a slice of `fmTx` (which we will define soon) and `maxTx`, the maximum number of transactions allowed in the mempool.

### Step 3: Check Interface Implementation

Before moving forward, let's ensure that our `FeeMempool` struct correctly implements the `mempool.Mempool` interface. We do this with the following line:

```go
var _ mempool.Mempool = (*FeeMempool)(nil)
```

If `FeeMempool` does not satisfy the `mempool.Mempool` interface, the Go compiler will throw an error.

### Step 4: Define

Now we will define `fmTx` struct and `fmTxs` type:

```go
type fmTx struct {
	address  string
	priority int64
	tx       sdk.Tx
}

var _ mempool.Iterator = fmTxs{}

type fmTxs []fmTx
```

We also need to define methods for the `fmTxs` type to satisfy the `mempool.Iterator` interface:

```go
var _ mempool.Iterator = fmTxs{}

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

```

Again, we do a compile-time check to ensure that `fmTxs` satisfies the `mempool.Iterator` interface.

### Step 5: Define FeeMempool Methods

Now we need to define methods for the `FeeMempool` type. These methods implement the `mempool.Mempool` interface, including `Insert`, `Select`, `CountTx` and `Remove`.

```go
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
		priority = naiveGetTxPriority(feeTx.GetFee(), int64(feeTx.GetGas())) // we will complete naiveGetTxPriority in the next steps
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
```

### Step 6: Implement NewFeeMempool Constructor

Finally, implement the constructor function `NewFeeMempool` which returns an instance of `FeeMempool`:

```go
func NewFeeMempool(logger log.Logger) *FeeMempool {
    return &FeeMempool{logger: logger.With("module", "fee-mempool")}
}

```

### Step 7: Define Helper Functions

Now we define `naiveGetTxPriority`, a helper function that is used to determine the priority of a transaction:

```go
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

```
This function takes the fee and gas of a transaction as inputs. It calculates the priority of a transaction by finding the smallest gas price (fee/gas) among all denominations in the fee. The calculated priority is then used when inserting the transaction into the mempool.

### Step 8: Use Helper Function in Mempool Methods

Now that you have the `naiveGetTxPriority` function, you can use it in the `Insert` method of `FeeMempool` to assign a priority to each transaction based on its fee and gas. This priority is then used to sort the transactions in the mempool.

With this, your `FeeMempool` is complete. The `naiveGetTxPriority` function is an important part of the `FeeMempool` as it allows transactions to be sorted and prioritized based on their fees.


### Step 9: Testing

Executing the genesis setup

```bash
make init
```

Starting the app and using the mempool type `sender nonce`

```bash
make install && minid start --mempool-type sender-nonce
```

Run send transaction script which will give us the mempool order.

```bash
./scripts/send_txs.sh
```

When using `--mempool-type none` when running the above script, it should return 
`CAROL - ALICE - BOB`

However, with the fee mempool `--mempool-type fee` the transactions or ordered by fees, so in the block it will be ordered as: `BOB - ALICE - CAROL`

And lastly with `--mempool-type sender-nonce` the transactions or ordered by the lowest nonce, so in the block it will be ordered as:
`CAROL - BOB - ALICE`


