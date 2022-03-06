package integration

import (
	"crypto/rsa"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"testing"
	"time"
)

func Test_three_nodes(t *testing.T) {
	log.Warn().Msg("=========================PHASE 1==================================")

	studentTransp := udpFac()
	chunkSize := uint(2)

	studentOpts := []z.Option{
		z.WithChunkSize(chunkSize),
		z.WithHeartbeat(time.Millisecond * 300),
		z.WithAntiEntropy(time.Second * 5),
		z.WithAckTimeout(time.Second * 10),
	}

	nodeA := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0", studentOpts...)
	nodeB := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0", studentOpts...)
	nodeC := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0", studentOpts...)
	nodeD := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0", studentOpts...)

	defer func() {
		nodeA.Stop()
		nodeB.Stop()
		nodeC.Stop()
		nodeD.Stop()
	}()

	log.Warn().Msg(" nodeA "+nodeA.GetAddr())
	log.Warn().Msg(" nodeB "+nodeB.GetAddr())
	log.Warn().Msg(" nodeC "+nodeC.GetAddr())
	log.Warn().Msg(" nodeD "+nodeD.GetAddr())

	nodeA.AddPeer(nodeB.GetAddr())
	//nodeA.AddPeer(nodeC.GetAddr())

	//nodeB.AddPeer(nodeA.GetAddr())
	nodeB.AddPeer(nodeC.GetAddr())

	//nodeC.AddPeer(nodeA.GetAddr())
	//nodeC.AddPeer(nodeB.GetAddr())

	nodeD.AddPeer(nodeA.GetAddr())

	defer func() {
		nodeA.Start()
		nodeB.Start()
		nodeC.Start()
		nodeD.Start()
	}()

	time.Sleep(time.Second * 10)
	log.Warn().Msg("=========================PHASE 2==================================")
	// Check if someone wants to change the pk of someone
	// E and F are alone first
	// E and F is going to have fake value for A

	nodeE := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0", studentOpts...)
	nodeF := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0", studentOpts...)

	defer func() {
		nodeE.Stop()
		nodeF.Stop()
	}()

	log.Warn().Msg(" nodeE "+nodeE.GetAddr())
	log.Warn().Msg(" nodeF "+nodeF.GetAddr())

	nodeE.AddPeer(nodeF.GetAddr())
	nodeF.AddPeer(nodeE.GetAddr())

	defer func() {
		nodeE.Start()
		nodeF.Start()
	}()

	time.Sleep(time.Second * 3)
	badTable := types.PublicKeyTable{}
	badTable[nodeA.GetAddr()] = rsa.PublicKey{}
	publicKeyMessage := types.PublicKeyMessage{
		TablePublicKey: badTable,
	}
	data, err := json.Marshal(&publicKeyMessage)
	require.NoError(t, err)

	msg := transport.Message{
		Type:    publicKeyMessage.Name(),
		Payload: data,
	}

	nodeE.Unicast(nodeF.GetAddr(),msg)

	time.Sleep(time.Second * 5)
	log.Warn().Msg("=========================PHASE 3==================================")
	nodeE.AddPeer(nodeC.GetAddr())
	nodeE.AddPeer(nodeB.GetAddr())
	nodeE.AddPeer(nodeD.GetAddr())

	nodeC.AddPeer(nodeE.GetAddr())

	time.Sleep(time.Second * 20)
}
