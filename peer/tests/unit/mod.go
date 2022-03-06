package unit

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
)

var peerFac peer.Factory = impl.NewPeer
