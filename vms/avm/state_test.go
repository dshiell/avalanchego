// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/avm/txs"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

func TestSetsAndGets(t *testing.T) {
	require := require.New(t)

	env := setup(t, &envConfig{
		additionalFxs: []*common.Fx{{
			ID: ids.GenerateTestID(),
			Fx: &FxTest{
				InitializeF: func(vmIntf interface{}) error {
					vm := vmIntf.(secp256k1fx.VM)
					return vm.CodecRegistry().RegisterType(&avax.TestState{})
				},
			},
		}},
	})
	defer func() {
		require.NoError(env.vm.Shutdown(context.Background()))
		env.vm.ctx.Lock.Unlock()
	}()

	utxo := &avax.UTXO{
		UTXOID: avax.UTXOID{
			TxID:        ids.Empty,
			OutputIndex: 1,
		},
		Asset: avax.Asset{ID: ids.Empty},
		Out:   &avax.TestState{},
	}
	utxoID := utxo.InputID()

	tx := &txs.Tx{Unsigned: &txs.BaseTx{BaseTx: avax.BaseTx{
		NetworkID:    constants.UnitTestID,
		BlockchainID: chainID,
		Ins: []*avax.TransferableInput{{
			UTXOID: avax.UTXOID{
				TxID:        ids.Empty,
				OutputIndex: 0,
			},
			Asset: avax.Asset{ID: assetID},
			In: &secp256k1fx.TransferInput{
				Amt: 20 * units.KiloAvax,
				Input: secp256k1fx.Input{
					SigIndices: []uint32{
						0,
					},
				},
			},
		}},
	}}}
	require.NoError(tx.SignSECP256K1Fx(env.vm.parser.Codec(), [][]*secp256k1.PrivateKey{{keys[0]}}))

	txID := tx.ID()

	env.vm.state.AddUTXO(utxo)
	env.vm.state.AddTx(tx)
	env.vm.state.AddStatus(txID, choices.Accepted)

	resultUTXO, err := env.vm.state.GetUTXO(utxoID)
	require.NoError(err)
	resultTx, err := env.vm.state.GetTx(txID)
	require.NoError(err)
	resultStatus, err := env.vm.state.GetStatus(txID)
	require.NoError(err)

	require.Equal(uint32(1), resultUTXO.OutputIndex)
	require.Equal(tx.ID(), resultTx.ID())
	require.Equal(choices.Accepted, resultStatus)
}

func TestFundingNoAddresses(t *testing.T) {
	env := setup(t, &envConfig{
		additionalFxs: []*common.Fx{{
			ID: ids.GenerateTestID(),
			Fx: &FxTest{
				InitializeF: func(vmIntf interface{}) error {
					vm := vmIntf.(secp256k1fx.VM)
					return vm.CodecRegistry().RegisterType(&avax.TestState{})
				},
			},
		}},
	})
	defer func() {
		require.NoError(t, env.vm.Shutdown(context.Background()))
		env.vm.ctx.Lock.Unlock()
	}()

	utxo := &avax.UTXO{
		UTXOID: avax.UTXOID{
			TxID:        ids.Empty,
			OutputIndex: 1,
		},
		Asset: avax.Asset{ID: ids.Empty},
		Out:   &avax.TestState{},
	}

	env.vm.state.AddUTXO(utxo)
	env.vm.state.DeleteUTXO(utxo.InputID())
}

func TestFundingAddresses(t *testing.T) {
	require := require.New(t)

	env := setup(t, &envConfig{
		additionalFxs: []*common.Fx{{
			ID: ids.GenerateTestID(),
			Fx: &FxTest{
				InitializeF: func(vmIntf interface{}) error {
					vm := vmIntf.(secp256k1fx.VM)
					return vm.CodecRegistry().RegisterType(&avax.TestAddressable{})
				},
			},
		}},
	})
	defer func() {
		require.NoError(env.vm.Shutdown(context.Background()))
		env.vm.ctx.Lock.Unlock()
	}()

	utxo := &avax.UTXO{
		UTXOID: avax.UTXOID{
			TxID:        ids.Empty,
			OutputIndex: 1,
		},
		Asset: avax.Asset{ID: ids.Empty},
		Out: &avax.TestAddressable{
			Addrs: [][]byte{{0}},
		},
	}

	env.vm.state.AddUTXO(utxo)
	require.NoError(env.vm.state.Commit())

	utxos, err := env.vm.state.UTXOIDs([]byte{0}, ids.Empty, math.MaxInt32)
	require.NoError(err)
	require.Len(utxos, 1)
	require.Equal(utxo.InputID(), utxos[0])

	env.vm.state.DeleteUTXO(utxo.InputID())
	require.NoError(env.vm.state.Commit())

	utxos, err = env.vm.state.UTXOIDs([]byte{0}, ids.Empty, math.MaxInt32)
	require.NoError(err)
	require.Empty(utxos)
}
