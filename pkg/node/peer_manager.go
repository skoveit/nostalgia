package node

import (
	"nostaliga/pkg/logger"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerManager struct {
	peers    map[peer.ID]time.Time
	maxPeers int
	mu       sync.RWMutex
}

func NewPeerManager(maxPeers int) *PeerManager {
	return &PeerManager{
		peers:    make(map[peer.ID]time.Time),
		maxPeers: maxPeers,
	}
}

func (pm *PeerManager) Add(p peer.ID) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[p]; exists {
		return false
	}

	if len(pm.peers) >= pm.maxPeers {
		return false
	}

	pm.peers[p] = time.Now()
	logger.Debug("→ Peer connected [%d/%d]: %s", len(pm.peers), pm.maxPeers, p.String())
	return true
}

func (pm *PeerManager) Remove(p peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[p]; exists {
		delete(pm.peers, p)
		logger.Debug("← Peer disconnected [%d/%d]: %s", len(pm.peers), pm.maxPeers, p.String())
	}
}

func (pm *PeerManager) Has(p peer.ID) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, exists := pm.peers[p]
	return exists
}

func (pm *PeerManager) IsFull() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.peers) >= pm.maxPeers
}

func (pm *PeerManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.peers)
}

func (pm *PeerManager) List() []peer.ID {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]peer.ID, 0, len(pm.peers))
	for p := range pm.peers {
		peers = append(peers, p)
	}
	return peers
}