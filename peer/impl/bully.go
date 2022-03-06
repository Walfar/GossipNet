package impl

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
	"time"
)

func (n *Node) GetCoordinator() uint {
	n.bully.RLock()
	defer n.bully.RUnlock()
	return n.bully.coordinator
}

func (n *Node) ConfigurePeers(network map[uint]string) {
	if len(network) <= 0 {
		return
	}
	newNetwork := make(map[uint]string)
	for k, v := range network {
		newNetwork[k] = v
	}
	n.bully.Lock()
	n.bully.network = newNetwork
	n.bully.Unlock()
}

func (n *Node) SetCoordinator(coordinator uint) {
	n.bully.Lock()
	n.bully.coordinator = coordinator
	n.bully.Unlock()
}

func (n *Node) Elect() {
	network := n.GetPeers()
	electionMessage := types.BullyMessage{
		PaxosID: n.config.PaxosID,
		Type:    types.BullyElection,
	}
	for k, v := range network {
		if k <= n.config.PaxosID {
			continue
		}
		Logger.Warn().Msgf("[%v] Send to %v", n.address, v)
		n.sendMessageUnchecked(v, electionMessage)
	}
	select {
	case <-n.bully.electionFailed:
		// We are not the coordinator
		Logger.Warn().Msgf("[%v] I am not the coordinator.", n.address)
		return
	case <-time.After(peer.BullyElectionTimeout):
		// No one replies to the election message after timeout
		// So I am going to be the coordinator
		Logger.Warn().Msgf("[%v] Set coordinator to: %v.", n.address, n.config.PaxosID)
		n.SetCoordinator(n.config.PaxosID)
		coordinatorMessage := types.BullyMessage{
			PaxosID: n.config.PaxosID,
			Type:    types.BullyCoordinator,
		}
		for _, v := range network {
			n.sendMessageUnchecked(v, coordinatorMessage)
		}
		return
	}
}

func (n *Node) GetPeers() map[uint]string {
	n.bully.RLock()
	defer n.bully.RUnlock()
	networkCopy := make(map[uint]string)
	for k, v := range n.bully.network {
		networkCopy[k] = v
	}
	return networkCopy
}

// ExecBullyMessage implements the handler for types.BullyMessage
func (n *Node) ExecBullyMessage(msg types.Message, pkt transport.Packet) error {
	bullyMessage, ok := msg.(*types.BullyMessage)
	if !ok {
		return xerrors.Errorf("Wrong type: %T", msg)
	}
	Logger.Warn().Msgf("[%v] ExecBullyMessage id=%v: receive bully message: %v", n.address, pkt.Header.PacketID, bullyMessage)
	switch bullyMessage.Type {
	case types.BullyElection:
		if bullyMessage.PaxosID < n.config.PaxosID {
			Logger.Warn().Msgf("[%v] %v Tries to become the coordinator, but should not be", n.address, bullyMessage.PaxosID)
			answerMessage := types.BullyMessage{
				PaxosID: n.config.PaxosID,
				Type:    types.BullyAnswer,
			}
			n.sendMessageUnchecked(pkt.Header.Source, answerMessage)
			go n.Elect()
		}
	case types.BullyAnswer:
		// Here it means that the election is failed
		n.bully.electionFailed <- peer.BullyElectionFailed
	case types.BullyCoordinator:
		Logger.Warn().Msgf("[%v] Set coordinator to: %v.", n.address, n.config.PaxosID)
		n.SetCoordinator(bullyMessage.PaxosID)
	}
	return nil
}
