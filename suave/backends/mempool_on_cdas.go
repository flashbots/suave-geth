package backends

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type MempoolOnConfidentialStore struct {
	cs suave.ConfidentialStoreBackend
}

func NewMempoolOnConfidentialStore(cs suave.ConfidentialStoreBackend) *MempoolOnConfidentialStore {
	return &MempoolOnConfidentialStore{
		cs: cs,
	}
}

func (m *MempoolOnConfidentialStore) Start() error {
	err := m.cs.InitializeBid(mempoolConfidentialStoreBid)
	if err != nil && !errors.Is(err, suave.ErrBidAlreadyPresent) {
		return fmt.Errorf("mempool: could not initialize: %w", err)
	}

	return nil
}

func (m *MempoolOnConfidentialStore) Stop() error {
	return nil
}

var (
	mempoolConfStoreId          = types.BidId{0x39}
	mempoolConfStoreAddr        = common.HexToAddress("0x39")
	mempoolConfidentialStoreBid = suave.Bid{Id: mempoolConfStoreId, AllowedPeekers: []common.Address{mempoolConfStoreAddr}}
)

func (m *MempoolOnConfidentialStore) SubmitBid(bid types.Bid) error {
	defer log.Info("bid submitted", "bid", bid, "store", m.cs.Store)

	var bidsByBlockAndProtocol []types.Bid
	bidsByBlockAndProtocolBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", bid.Version, bid.DecryptionCondition))
	if err == nil {
		bidsByBlockAndProtocol = suave.MustDecode[[]types.Bid](bidsByBlockAndProtocolBytes)
	}
	// store bid by block number and by protocol + block number
	bidsByBlockAndProtocol = append(bidsByBlockAndProtocol, bid)

	m.cs.Store(mempoolConfidentialStoreBid, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", bid.Version, bid.DecryptionCondition), suave.MustEncode(bidsByBlockAndProtocol))

	return nil
}

func (m *MempoolOnConfidentialStore) FetchBidById(bidId suave.BidId) (types.Bid, error) {
	engineBid, err := m.cs.FetchEngineBidById(bidId)
	if err != nil {
		log.Error("bid missing!", "id", bidId, "err", err)
		return types.Bid{}, errors.New("not found")
	}

	return types.Bid{
		Id:                  engineBid.Id,
		Salt:                engineBid.Salt,
		DecryptionCondition: engineBid.DecryptionCondition,
		AllowedPeekers:      engineBid.AllowedPeekers,
		AllowedStores:       engineBid.AllowedStores,
		Version:             engineBid.Version,
	}, nil
}

func (m *MempoolOnConfidentialStore) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []types.Bid {
	bidsByProtocolBytes, err := m.cs.Retrieve(mempoolConfidentialStoreBid, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", namespace, blockNumber))
	if err != nil {
		return nil
	}
	defer log.Info("bids fetched", "bids", string(bidsByProtocolBytes))
	return suave.MustDecode[[]types.Bid](bidsByProtocolBytes)
}
