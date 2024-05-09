package suavesdk

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// lets keep it simple and add a global SuaveSDK instance
var sdk *SuaveSDK

func GetSDK() *SuaveSDK {
	return sdk
}

func Start() {
	if sdk != nil {
		panic("SDK already started")
	}

	var err error
	sdk, err = New()
	if err != nil {
		panic(err)
	}
}

// DiscoveryServiceTag is used in our mDNS advertisements to discover other suave peers.
const DiscoveryServiceTag = "suave-lfg"

type SuaveSDK struct {
	node host.Host
	ps   *pubsub.PubSub

	// we have to keep a map of topics because ps.Join fails if we try
	// to subscribe to the same topic multiple times.
	topics map[string]*pubsub.Topic
}

func New() (*SuaveSDK, error) {
	// create a new libp2p Host that listens on a random TCP port
	node, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, err
	}

	// print the node's listening addresses
	for _, addr := range node.Addrs() {
		log.Info("Suave-sdk, Listening on", "addr", addr)
	}

	node.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			log.Info("Suave-sdk, Connected to", "peer", c.RemotePeer())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			log.Info("Suave-sdk, Disconnected from", "peer", c.RemotePeer())
		},
	})

	// create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(context.TODO(), node)
	if err != nil {
		return nil, err
	}

	// setup local mDNS discovery
	if err := setupDiscovery(node); err != nil {
		panic(err)
	}

	return &SuaveSDK{
		node:   node,
		ps:     ps,
		topics: make(map[string]*pubsub.Topic),
	}, nil
}

func (s *SuaveSDK) Close() {
	if err := s.node.Close(); err != nil {
		panic(err)
	}
}

func (s *SuaveSDK) Topic(name string) (*pubsub.Topic, error) {
	if topic, ok := s.topics[name]; ok {
		return topic, nil
	}

	topic, err := s.ps.Join(name)
	if err != nil {
		return nil, err
	}

	s.topics[name] = topic
	return topic, nil
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Info("Discovered SUAVE new peer", "id", pi.ID.String())

	// add some random delay of n secodns if your id is lower than the peer id, otherwise, there might
	// be TCP problems when connecting at the same time.
	if n.h.ID().String() < pi.ID.String() {
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	}

	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID, err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})
	return s.Start()
}
