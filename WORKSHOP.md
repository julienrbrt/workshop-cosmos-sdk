# Workshop

## Introduction

Cosmos SDK v0.47 implements ABCI++ with [Prepare and Process Proposal](https://docs.cometbft.com/v0.37/spec/abci/abci++_basic_concepts.html).
These ABCI methods allow the developer to completely take control of the block creation.
This means developers can easily decide what tx are prioritized, what a block should contain and verify transactions before them being included in a block. Cosmos SDK Twilight facilitates that by providing a `mempool.Mempool` interface package that allows developers to create their own mempool.

In this workshop, we will create a mempool that prioritizes transactions with the highest fee.
We will use the default Process and Prepare Proposal handlers of the Cosmos SDK, where when a mempool is used, only valid transactions are included in a block.
Note, that the SDK provides by default 3 mempools: Sender Nonce, Sender Nonce Priority and No-Op.

* The No-Op mempool keeps the same behavior as the previous SDK versions, and processes transactions in the order they are in CometBFT mempool.
* Sender Nonce is a mempool that prioritizes transactions within a sender by nonce, the lowest first, but selects a random sender on each iteration.
* Sender Nonce Priority (Fee) is a mempool implementation that stores txs in a partially ordered set by 2 dimensions: priority, and sender-nonce (sequence number).

The default mempool of the SDK is the No-Op mempool, to keep the behavior of the previous SDK versions.

## How to set a mempool in a Cosmos SDK chain

A mempool implementing the `mempool.Mempool` interface, can simply be added to a Cosmos SDK chain by using the baseapp `SetMempool` method in the `app.go` file.

```go
mempoolOpt := func(app *baseapp.BaseApp) {
	app.SetMempool(selectedMempool)
}
```

## Creating a Fee Mempool

As mentionned earlier, to create a mempool, we need to implement the `mempool.Mempool` interface.

The interface is defined as follow:

```go
type Mempool interface {
	// Insert attempts to insert a Tx into the app-side mempool returning
	// an error upon failure.
	Insert(context.Context, sdk.Tx) error

	// Select returns an Iterator over the app-side mempool. If txs are specified,
	// then they shall be incorporated into the Iterator. The Iterator must
	// closed by the caller.
	Select(context.Context, [][]byte) Iterator

	// CountTx returns the number of transactions currently in the mempool.
	CountTx() int

	// Remove attempts to remove a transaction from the mempool, returning an error
	// upon failure.
	Remove(sdk.Tx) error
}
```

### Step 1: Define FeeMempool

Next, we define a `FeeMempool` struct that will store the transactions.

```go
type FeeMempool struct {
	logger log.Logger
	pool   fmTxs
}
```

Before moving forward, let's ensure that our `FeeMempool` struct correctly implements the `mempool.Mempool` interface.

```go
var _ mempool.Mempool = (*FeeMempool)(nil)
```

If `FeeMempool` does not satisfy the `mempool.Mempool` interface, the Go compiler will throw an error.


### Step 2: Implement interface methods

Implement on the `FeeMempool` struct the different mempool methods.

See [fee.go](./mempool/fee.go) for example.

### Step 3: Testing

For this workshop we have provided a few helpers to easily run an application and test it.

Execute the genesis setup (creates 3 accounts, and gives them some tokens and a validator):

```bash
make install
make init
```

The application requires a `--mempool-type` flag to be set, so we can test the different mempools.
Normally, this flag isn't required in an actual application.

Start the node with the following command:

```bash
minid start --mempool-type [mempool-type]
```

You can use the SDK mempools (`none` (i.e NoOp), `sender-nonce` or `priority-nonce`).
Additionally you can use the fee mempool we have created here with `fee` or use you own (not you'll need to add it to the `app.go` file)

Finally, run some tests transactions and see how the mempool orders them.

```bash
./scripts/send_txs.sh
```

When using `--mempool-type none` when running the above script, it should return `CAROL - ALICE - BOB`.
However, with the fee mempool `--mempool-type fee` the transactions are ordered by fees, so in the block it will be ordered as: `BOB - ALICE - CAROL`.

The other mempool are ordered randomly (but determinastically thanks to the seed). Try it out to see what you get.
