package integration

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/udp"
)

var studentFac peer.Factory = impl.NewPeer

var udpFac transport.Factory = udp.NewUDP
