package node

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"

	"nostaliga/pkg/logger"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const MaxPeers = 5

type Protocol interface {
	HandleStream(network.Stream)
}

type Node struct {
	host      host.Host
	ctx       context.Context
	peerMgr   *PeerManager
	protocol  Protocol
	protoLock sync.RWMutex
}

func NewNode(ctx context.Context) (*Node, error) {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	n := &Node{
		host:    h,
		ctx:     ctx,
		peerMgr: NewPeerManager(MaxPeers),
	}

	h.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(_ network.Network, c network.Conn) {
			n.peerMgr.Remove(c.RemotePeer())
		},
	})

	return n, nil
}

func (n *Node) SetProtocol(p Protocol) {
	n.protoLock.Lock()
	n.protocol = p
	n.protoLock.Unlock()
}

func (n *Node) Host() host.Host {
	return n.host
}

func (n *Node) ID() peer.ID {
	return n.host.ID()
}

func (n *Node) Addrs() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

func (n *Node) Context() context.Context {
	return n.ctx
}

func (n *Node) PeerManager() *PeerManager {
	return n.peerMgr
}

// ListPeers logs peer list (for debug) and returns formatted string
func (n *Node) ListPeers() string {
	peers := n.peerMgr.List()
	if len(peers) == 0 {
		logger.Debugln("No connected peers")
		return "No connected peers"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Connected peers (%d/%d):\n", len(peers), MaxPeers))
	for _, p := range peers {
		sb.WriteString(fmt.Sprintf("  - %s\n", p.String()))
	}

	result := sb.String()
	logger.Debug("%s", result)
	return result
}
