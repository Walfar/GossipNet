package peer

import "time"

const BullyElectionTimeout = time.Second
const BullyElectionFailed = "Failed"

type Bully interface {

	// GetCoordinator returns the current coordinator in the network
	GetCoordinator() uint

	// ConfigurePeers sets up the network so that the peer is aware of its ID and others IDs
	ConfigurePeers(network map[uint]string)

	// SetCoordinator sets the coordinator in the network
	SetCoordinator(coordinator uint)

	// Elect tries to elect the current node as the coordinator
	Elect()

	// GetPeers returns the network for bully algorithm
	GetPeers() map[uint]string
}
