package txpool

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suavesdk"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type BundlePool struct {
	lock    sync.Mutex
	bundles []*types.MossBundle
	topic   *pubsub.Topic
}

func NewBundlePool() *BundlePool {
	return &BundlePool{
		bundles: make([]*types.MossBundle, 0),
	}
}

func (b *BundlePool) StartP2P() *BundlePool {
	// connect to the suavesdk network and listen for bundles
	sdk := suavesdk.GetSDK()
	if sdk == nil {
		return b
	}

	topic, err := sdk.Topic("moss-bundle")
	if err != nil {
		panic(err)
	}
	b.topic = topic

	sub, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			msg, err := sub.Next(context.TODO())
			if err != nil {
				panic(err)
			}

			bundle := &types.MossBundle{}
			if err = json.Unmarshal(msg.Data, &bundle); err != nil {
				panic(err)
			}

			b.AddBundle(bundle)
		}
	}()

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

func (b *BundlePool) AddLocalBundle(bundle *types.MossBundle) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.bundles = append(b.bundles, bundle)
}

func (b *BundlePool) AddBundle(bundle *types.MossBundle) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.bundles = append(b.bundles, bundle)

	// relay it to the p2p network
	data, err := json.Marshal(bundle)
	if err != nil {
		panic(err)
	}
	if b.topic != nil {
		if err := b.topic.Publish(context.TODO(), data); err != nil {
			panic(err)
		}
	}
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
