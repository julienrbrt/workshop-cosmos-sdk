package mempool_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/require"

	"github.com/julienrbrt/chain-minimal/mempool"
)

func TestTxOrder(t *testing.T) {
	accounts := simtypes.RandomAccounts(rand.New(rand.NewSource(0)), 5)
	sa := accounts[0].Address
	sb := accounts[1].Address
	sc := accounts[2].Address
	sd := accounts[3].Address

	tests := []struct {
		txs   []txSpec
		order []int
	}{
		{
			txs: []txSpec{
				{sender: sa, priority: 0},
				{sender: sb, priority: 20},
				{sender: sc, priority: 30},
				{sender: sd, priority: 50},
			},
			order: []int{3, 2, 1, 0},
		},
		{
			txs: []txSpec{
				{sender: sa, priority: 30},
				{sender: sb, priority: 10},
				{sender: sa, priority: 20},
				{sender: sd, priority: 0},
			},
			order: []int{0, 2, 1, 3},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			pool := mempool.NewFeeMempool(log.TestingLogger())
			// create test txs and insert into mempool
			for i, ts := range tt.txs {
				tx := testTx{id: i, priority: int64(ts.priority), address: ts.sender, nonce: uint64(i)}
				err := pool.Insert(context.Background(), tx)
				require.NoError(t, err)
				require.Equal(t, i+1, pool.CountTx())
			}

			var orderedTxs []sdk.Tx
			itr := pool.Select(context.Background(), nil)
			for itr != nil {
				orderedTxs = append(orderedTxs, itr.Tx())
				itr = itr.Next()
			}

			var txOrder []int
			for _, tx := range orderedTxs {
				txOrder = append(txOrder, tx.(testTx).id)
			}
			for _, tx := range orderedTxs {
				require.NoError(t, pool.Remove(tx))
			}
			require.Equal(t, tt.order, txOrder)
			require.Equal(t, 0, pool.CountTx())
		})
	}
}
