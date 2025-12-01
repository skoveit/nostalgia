package discovery

import "github.com/libp2p/go-libp2p/core/peer"

type Discovery interface {
	Start() error
	Stop() error
	OnPeerFound(peer.AddrInfo)
}
