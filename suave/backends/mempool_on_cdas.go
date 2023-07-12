package backends

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type MempoolOnConfidentialStore struct {
	cs suave.ConfiendialStoreBackend
}

func NewMempoolOnConfidentialStore(cs suave.ConfiendialStoreBackend) *MempoolOnConfidentialStore {
	_, err := cs.Initialize(mempoolConfidentialStoreBid, "", nil)
	if err != nil {
		panic("could not initialize mempool")
	}
	return &MempoolOnConfidentialStore{
		cs: cs,
	}
}

var (
	mempoolConfStoreId          = [16]byte{0x39}
	mempoolConfStoreAddr        = common.HexToAddress("0x39")
	mempoolConfidentialStoreBid = suave.Bid{Id: mempoolConfStoreId, AllowedPeekers: []common.Address{mempoolConfStoreAddr}}
)

func (m *MempoolOnConfidentialStore) SubmitBid(bid suave.Bid) error {
	m.cs.Store(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("id-%x", bid.Id), suave.MustEncode(bid))

	var bidsByBlockNumber []suave.Bid
	bidsByBlockNumberBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("bn-%d", bid.DecryptionCondition))
	if err == nil {
		bidsByBlockNumber = suave.MustDecode[[]suave.Bid](bidsByBlockNumberBytes)
	}
	// store bid by block number and by protocol + block number
	bidsByBlockNumber = append(bidsByBlockNumber, bid)
	m.cs.Store(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("bn-%d", bid.DecryptionCondition), suave.MustEncode(bidsByBlockNumber))

	m.cs.Store(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", bid.Version, bid.DecryptionCondition), suave.MustEncode(bidsByBlockNumber))

	return nil
}

func (m *MempoolOnConfidentialStore) FetchBids(blockNumber uint64) []suave.Bid {
	bidsByBlockNumberBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("bn-%d", blockNumber))
	if err != nil {
		return nil
	}
	return suave.MustDecode[[]suave.Bid](bidsByBlockNumberBytes)
}

func (m *MempoolOnConfidentialStore) FetchBidById(bidId suave.BidId) (suave.Bid, error) {
	bidBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("id-%x", bidId))
	if err != nil {
		return suave.Bid{}, errors.New("not found")
	}
	return suave.MustDecode[suave.Bid](bidBytes), nil
}

func (m *MempoolOnConfidentialStore) FetchBidsByBlock(blockNumber uint64) []suave.Bid {
	bidsByBlockNumberBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("bn-%d", blockNumber))
	if err != nil {
		return nil
	}
	return suave.MustDecode[[]suave.Bid](bidsByBlockNumberBytes)
}

func (m *MempoolOnConfidentialStore) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	bidsByProtocolBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid.Id, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", namespace, blockNumber))
	if err != nil {
		return nil
	}
	return suave.MustDecode[[]suave.Bid](bidsByProtocolBytes)
}
