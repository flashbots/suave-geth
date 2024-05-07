package txpool

import (
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
)

type BundlePool struct {
	lock    sync.Mutex
	bundles []*types.MossBundle
}

func NewBundlePool() *BundlePool {
	return &BundlePool{
		bundles: make([]*types.MossBundle, 0),
	}
}

func (b *BundlePool) StartP2P() *BundlePool {
	// connect to the suavesdk network and listen for bundles
	return b
}

func (b *BundlePool) ResetPool(head *types.Header) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// remove all the bundles that have passed their inclusion block
	for _, bundle := range b.bundles {
		if head.Number.Uint64() >= bundle.MaxBlockNumber {
			b.bundles = append(b.bundles[:0], b.bundles[1:]...)
		}
	}
}

func (b *BundlePool) AddBundle(bundle *types.MossBundle) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.bundles = append(b.bundles, bundle)
}

func (b *BundlePool) GetBundles(blockNum uint64) []*types.MossBundle {
	b.lock.Lock()
	defer b.lock.Unlock()

	result := []*types.MossBundle{}
	for _, bundle := range b.bundles {
		if blockNum >= bundle.BlockNumber && blockNum < bundle.MaxBlockNumber {
			result = append(result, bundle.Copy())
		}
	}
	return result
}
