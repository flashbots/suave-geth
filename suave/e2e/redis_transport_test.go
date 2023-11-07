package e2e

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/stretchr/testify/require"
)

func TestRedisBackends(t *testing.T) {
	withMiniredisTransportOpt := WithRedisTransportOpt(t)

	fr1 := newFramework(t, WithKettleAddress(), WithRedisStoreBackend(), withMiniredisTransportOpt)
	t.Cleanup(fr1.Close)

	var keystoreBackend *keystore.KeyStore = fr1.suethSrv.service.APIBackend.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	keystoreBackend.ImportECDSA(testKey, "")

	fr2 := newFramework(t, WithKettleAddress(), WithRedisStoreBackend(), withMiniredisTransportOpt)
	t.Cleanup(fr2.Close)

	clt1 := fr1.NewSDKClient()
	clt2 := fr2.NewSDKClient()

	ethTx, err := clt1.SignTxn(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	})
	require.NoError(t, err)

	targetBlock := uint64(1)
	bundle := &types.SBundle{
		BlockNumber:     big.NewInt(int64(targetBlock)),
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{newBlockBidAddress, newBundleBidAddress, buildEthBlockAddress}

		confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		bundleBidContractI := sdk.GetContract(newBundleBidAddress, BundleBidContract.Abi, clt1)

		_, err = bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{fr1.KettleAddress(), fr2.KettleAddress()}}, confidentialDataBytes)
		requireNoRpcError(t, err)
	}

	block := fr1.suethSrv.ProgressChain()
	require.Equal(t, 1, len(block.Transactions()))

	time.Sleep(1000 * time.Millisecond)

	// TODO: explicitly check if fr2 received the bid

	{
		ethHead := fr2.ethSrv.CurrentBlock()

		payloadArgsTuple := types.BuildBlockArgs{
			ProposerPubkey: []byte{0x42},
			Timestamp:      ethHead.Time + uint64(12),
			FeeRecipient:   common.Address{0x42},
		}

		buildEthBlockContractI := sdk.GetContract(newBlockBidAddress, buildEthBlockContract.Abi, clt2)

		_, err = buildEthBlockContractI.SendTransaction("buildFromPool", []interface{}{payloadArgsTuple, targetBlock + 1}, nil)
		requireNoRpcError(t, err)

		block = fr2.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))
	}
}
