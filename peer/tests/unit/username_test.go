package unit

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/channel"
	"go.dedis.ch/cs438/types"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Unit test to see if:
// 		peer.BuildUsernamePaxosValue
// 			and
// 		peer.ParseUsernamePaxosValue
// are working as desired
func Test_Username_Build_And_Parse_Username(t *testing.T) {
	jsonWithoutOldName := `{"OldName":"","NewName":"new1"}`
	jsonWithOldName := `{"OldName":"new1","NewName":"new2"}`

	// Test Marshal
	v, err := peer.BuildUsernamePaxosValue("new1", "")
	require.NoError(t, err)
	require.Equal(t, jsonWithoutOldName, v)

	v, err = peer.BuildUsernamePaxosValue("new2", "new1")
	require.NoError(t, err)
	require.Equal(t, jsonWithOldName, v)

	// Test Unmarshal
	oldName, newName, err := peer.ParseUsernamePaxosValue(jsonWithoutOldName)
	require.NoError(t, err)
	require.Equal(t, "", oldName)
	require.Equal(t, "new1", newName)

	oldName, newName, err = peer.ParseUsernamePaxosValue(jsonWithOldName)
	require.NoError(t, err)
	require.Equal(t, "new1", oldName)
	require.Equal(t, "new2", newName)

	_, _, err = peer.ParseUsernamePaxosValue(randomString(20))
	require.Error(t, err)
}

// Test Node.SetUsername with a simple topology
// 		node1 -> node2
// First, set username of node1 to A
// 		-> Both node1 and node2 has one record in the naming store
//		   A: node1Address
// Second, set username of node2 to B
// 		-> Both node1 and node2 has two records in the naming store
//		   A: node1Address
//		   B: node2Address
// Third, set username of node1 to C
// 		-> Both node1 and node2 has two records in the naming store
//		   C: node1Address
//		   B: node2Address
// Fourth, set username of node2 to A (now node2 can reuse node1's old name)
// 		-> Both node1 and node2 has two records in the naming store
//		   C: node1Address
//		   A: node2Address
func Test_Username_SetUsername_Simple(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(2), z.WithPaxosID(1), z.WithDisabledPKI())
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(2), z.WithPaxosID(2), z.WithDisabledPKI())
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())

	node1NewName := "this is my new fancy name!"
	err := node1.SetUsername(node1NewName)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > node1 must have sent
	//
	//   - Rumor(1):PaxosPrepare
	//   - Rumor(2):Private:PaxosPromise
	//   - Rumor(3):PaxosPropose
	//   - Rumor(4):PaxosAccept
	//   - Rumor(5):TLC
	n1outs := node1.GetOuts()

	// >> Rumor(1):PaxosPrepare
	msg, pkt := getRumor(t, n1outs, 1)
	require.NotNil(t, msg)
	require.Equal(t, node1.GetAddr(), pkt.Source)
	require.Equal(t, node1.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node2.GetAddr(), pkt.Destination)
	prepare := z.GetPaxosPrepare(t, msg)
	require.Equal(t, uint(1), prepare.ID)
	require.Equal(t, uint(0), prepare.Step)
	require.Equal(t, node1.GetAddr(), prepare.Source)

	// >> Rumor(2):Private:PaxosPromise
	msg, pkt = getRumor(t, n1outs, 2)
	require.NotNil(t, msg)
	require.Equal(t, node1.GetAddr(), pkt.Source)
	require.Equal(t, node1.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node2.GetAddr(), pkt.Destination)
	private := z.GetPrivate(t, msg)
	require.Len(t, private.Recipients, 1)
	require.Contains(t, private.Recipients, node1.GetAddr())
	promise := z.GetPaxosPromise(t, private.Msg)
	require.Equal(t, uint(1), promise.ID)
	require.Equal(t, uint(0), promise.Step)
	// default "empty" value is 0
	require.Zero(t, promise.AcceptedID)
	require.Nil(t, promise.AcceptedValue)

	// >> Rumor(3):PaxosPropose
	msg, pkt = getRumor(t, n1outs, 3)
	require.NotNil(t, msg)
	require.Equal(t, node1.GetAddr(), pkt.Source)
	require.Equal(t, node1.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node2.GetAddr(), pkt.Destination)

	propose := z.GetPaxosPropose(t, msg)
	jsonValue, err := peer.BuildUsernamePaxosValue(node1NewName, "")
	require.NoError(t, err)
	require.Equal(t, uint(1), propose.ID)
	require.Equal(t, uint(0), propose.Step)
	require.Equal(t, jsonValue, propose.Value.Filename)
	require.Equal(t, node1.GetAddr(), propose.Value.Metahash)

	// >> Rumor(4):PaxosAccept
	msg, pkt = getRumor(t, n1outs, 4)
	require.NotNil(t, msg)
	require.Equal(t, node1.GetAddr(), pkt.Source)
	require.Equal(t, node1.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node2.GetAddr(), pkt.Destination)
	accept := z.GetPaxosAccept(t, msg)
	require.Equal(t, uint(1), accept.ID)
	require.Equal(t, uint(0), accept.Step)
	require.Equal(t, jsonValue, accept.Value.Filename)
	require.Equal(t, node1.GetAddr(), accept.Value.Metahash)

	// >> Rumor(5):TLC
	msg, pkt = getRumor(t, n1outs, 5)
	require.NotNil(t, msg)
	require.Equal(t, node1.GetAddr(), pkt.Source)
	require.Equal(t, node1.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node2.GetAddr(), pkt.Destination)
	tlc := z.GetTLC(t, msg)
	require.Equal(t, uint(0), tlc.Step)
	require.Equal(t, uint(0), tlc.Block.Index)
	require.Equal(t, make([]byte, 32), tlc.Block.PrevHash)
	require.Equal(t, jsonValue, tlc.Block.Value.Filename)
	require.Equal(t, node1.GetAddr(), tlc.Block.Value.Metahash)

	// > node2 must have sent
	//
	//   - Rumor(1):Private:PaxosPromise
	//   - Rumor(2):PaxosAccept
	//   - Rumor(3):TLC
	n2outs := node2.GetOuts()

	// >> Rumor(1):Private:PaxosPromise
	msg, pkt = getRumor(t, n2outs, 1)
	require.NotNil(t, msg)
	require.Equal(t, node2.GetAddr(), pkt.Source)
	require.Equal(t, node2.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node1.GetAddr(), pkt.Destination)
	private = z.GetPrivate(t, msg)
	require.Len(t, private.Recipients, 1)
	require.Contains(t, private.Recipients, node1.GetAddr())
	promise = z.GetPaxosPromise(t, private.Msg)
	require.Equal(t, uint(1), promise.ID)
	require.Equal(t, uint(0), promise.Step)
	// default "empty" value is 0
	require.Zero(t, promise.AcceptedID)
	require.Nil(t, promise.AcceptedValue)

	// >> Rumor(2):PaxosAccept
	msg, pkt = getRumor(t, n2outs, 2)
	require.NotNil(t, msg)
	require.Equal(t, node2.GetAddr(), pkt.Source)
	require.Equal(t, node2.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node1.GetAddr(), pkt.Destination)
	accept = z.GetPaxosAccept(t, msg)
	require.Equal(t, uint(1), accept.ID)
	require.Equal(t, uint(0), accept.Step)
	require.Equal(t, jsonValue, accept.Value.Filename)
	require.Equal(t, node1.GetAddr(), accept.Value.Metahash)

	// >> Rumor(3):TLC
	msg, pkt = getRumor(t, n2outs, 3)
	require.NotNil(t, msg)
	require.Equal(t, node2.GetAddr(), pkt.Source)
	require.Equal(t, node2.GetAddr(), pkt.RelayedBy)
	require.Equal(t, node1.GetAddr(), pkt.Destination)
	tlc = z.GetTLC(t, msg)
	require.Equal(t, uint(0), tlc.Step)
	require.Equal(t, uint(0), tlc.Block.Index)
	require.Equal(t, make([]byte, 32), tlc.Block.PrevHash)
	require.Equal(t, jsonValue, tlc.Block.Value.Filename)
	require.Equal(t, node1.GetAddr(), tlc.Block.Value.Metahash)

	// > node1 name store is updated
	names := node1.GetStorage().GetNamingStore()
	require.Equal(t, 1, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewName))

	// > node2 name store is updated
	names = node2.GetStorage().GetNamingStore()
	require.Equal(t, 1, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewName))

	// > node1 blockchain store contains two elements
	bstore := node1.GetStorage().GetBlockchainStore()
	require.Equal(t, 2, bstore.Len())
	lastBlockHash := bstore.Get(storage.LastBlockKey)
	lastBlock := bstore.Get(hex.EncodeToString(lastBlockHash))
	var block types.BlockchainBlock
	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)
	require.Equal(t, uint(0), block.Index)
	require.Equal(t, make([]byte, 32), block.PrevHash)
	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node1.GetAddr(), block.Value.Metahash)

	// > node2 blockchain store contains two elements
	bstore = node2.GetStorage().GetBlockchainStore()
	require.Equal(t, 2, bstore.Len())
	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))
	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)
	require.Equal(t, uint(0), block.Index)
	require.Equal(t, make([]byte, 32), block.PrevHash)
	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node1.GetAddr(), block.Value.Metahash)
	newNameFromNode, err := node1.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node1NewName, newNameFromNode)

	//////////////////////////////////////////////////////////////////////////
	// The above finishes setting the name for node1

	// Now if node2 tries to set the same name, there will be an error
	err = node2.SetUsername(node1NewName)
	require.Error(t, err)

	// Now if node1 tries to set it again, there will be no error
	err = node1.SetUsername(node1NewName)
	require.NoError(t, err)

	//////////////////////////////////////////////////////////////////////////
	// Now let's set node2 to another name
	node2NewName := "this is the new name for node 2"
	err = node2.SetUsername(node2NewName)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > node1 name store is updated
	names = node1.GetStorage().GetNamingStore()
	require.Equal(t, 2, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewName))
	require.Equal(t, []byte(node2.GetAddr()), names.Get(node2NewName))

	// > node2 name store is updated

	names = node2.GetStorage().GetNamingStore()
	require.Equal(t, 2, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewName))
	require.Equal(t, []byte(node2.GetAddr()), names.Get(node2NewName))

	// > node1 blockchain store contains three elements

	bstore = node1.GetStorage().GetBlockchainStore()

	require.Equal(t, 3, bstore.Len())

	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))

	jsonValue, err = peer.BuildUsernamePaxosValue(node2NewName, "")
	require.NoError(t, err)

	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)

	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node2.GetAddr(), block.Value.Metahash)

	// > node2 blockchain store contains three elements

	bstore = node2.GetStorage().GetBlockchainStore()

	require.Equal(t, 3, bstore.Len())

	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))

	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)

	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node2.GetAddr(), block.Value.Metahash)

	newNameFromNode, err = node1.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node1NewName, newNameFromNode)

	newNameFromNode, err = node2.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node2NewName, newNameFromNode)

	z.ValidateBlockchain(t, node1.GetStorage().GetBlockchainStore())
	z.ValidateBlockchain(t, node2.GetStorage().GetBlockchainStore())

	z.DisplayLastBlockchainBlock(t, os.Stdout, node1.GetStorage().GetBlockchainStore())
	z.DisplayLastBlockchainBlock(t, os.Stdout, node2.GetStorage().GetBlockchainStore())

	//////////////////////////////////////////////////////////////////////////
	// Now let's change again node1 to a different name
	node1NewNewName := "this is the latest name for node 1, yes!"
	err = node1.SetUsername(node1NewNewName)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > node1 name store is updated
	names = node1.GetStorage().GetNamingStore()
	require.Equal(t, 2, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewNewName))
	require.Equal(t, []byte(node2.GetAddr()), names.Get(node2NewName))

	// > node2 name store is updated
	names = node2.GetStorage().GetNamingStore()
	require.Equal(t, 2, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewNewName))
	require.Equal(t, []byte(node2.GetAddr()), names.Get(node2NewName))

	// > node1 blockchain store contains four elements
	bstore = node1.GetStorage().GetBlockchainStore()
	require.Equal(t, 4, bstore.Len())
	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))
	jsonValue, err = peer.BuildUsernamePaxosValue(node1NewNewName, node1NewName)
	require.NoError(t, err)
	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)
	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node1.GetAddr(), block.Value.Metahash)

	// > node2 blockchain store contains four elements
	bstore = node2.GetStorage().GetBlockchainStore()
	require.Equal(t, 4, bstore.Len())
	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))
	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)
	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node1.GetAddr(), block.Value.Metahash)

	newNameFromNode, err = node1.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node1NewNewName, newNameFromNode)

	newNameFromNode, err = node2.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node2NewName, newNameFromNode)

	z.ValidateBlockchain(t, node1.GetStorage().GetBlockchainStore())
	z.ValidateBlockchain(t, node2.GetStorage().GetBlockchainStore())

	z.DisplayLastBlockchainBlock(t, os.Stdout, node1.GetStorage().GetBlockchainStore())
	z.DisplayLastBlockchainBlock(t, os.Stdout, node2.GetStorage().GetBlockchainStore())

	//////////////////////////////////////////////////////////////////////////
	// Now let's reuse node1's name for node2
	err = node2.SetUsername(node1NewName)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > node1 name store is updated
	names = node1.GetStorage().GetNamingStore()
	require.Equal(t, 2, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewNewName))
	require.Equal(t, []byte(node2.GetAddr()), names.Get(node1NewName))

	// > node2 name store is updated
	names = node2.GetStorage().GetNamingStore()
	require.Equal(t, 2, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get(node1NewNewName))
	require.Equal(t, []byte(node2.GetAddr()), names.Get(node1NewName))

	// > node1 blockchain store contains five elements
	bstore = node1.GetStorage().GetBlockchainStore()
	require.Equal(t, 5, bstore.Len())
	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))
	jsonValue, err = peer.BuildUsernamePaxosValue(node1NewName, node2NewName)
	require.NoError(t, err)
	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)
	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node2.GetAddr(), block.Value.Metahash)

	// > node2 blockchain store contains five elements
	bstore = node2.GetStorage().GetBlockchainStore()
	require.Equal(t, 5, bstore.Len())
	lastBlockHash = bstore.Get(storage.LastBlockKey)
	lastBlock = bstore.Get(hex.EncodeToString(lastBlockHash))
	err = block.Unmarshal(lastBlock)
	require.NoError(t, err)
	require.Equal(t, jsonValue, block.Value.Filename)
	require.Equal(t, node2.GetAddr(), block.Value.Metahash)

	newNameFromNode, err = node1.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node1NewNewName, newNameFromNode)

	newNameFromNode, err = node2.GetUsername()
	require.NoError(t, err)
	require.Equal(t, node1NewName, newNameFromNode)

	z.ValidateBlockchain(t, node1.GetStorage().GetBlockchainStore())
	z.ValidateBlockchain(t, node2.GetStorage().GetBlockchainStore())

	z.DisplayLastBlockchainBlock(t, os.Stdout, node1.GetStorage().GetBlockchainStore())
	z.DisplayLastBlockchainBlock(t, os.Stdout, node2.GetStorage().GetBlockchainStore())
}

// Test Node.SetUsername can reach an eventual consensus
func Test_Username_SetUsername_Eventual_Consensus(t *testing.T) {
	transp := channel.NewTransport()

	threshold := func(i uint) int { return int(i) }

	// Note: we are setting the antientropy on each peer to make sure all rumors
	// are spread among peers.
	// Threshold = 3
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithTotalPeers(3),
		z.WithPaxosID(1),
		z.WithPaxosThreshold(threshold),
		z.WithPaxosProposerRetry(time.Second*2),
		z.WithAntiEntropy(time.Second), z.WithDisabledPKI())
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithTotalPeers(3),
		z.WithPaxosID(2),
		z.WithPaxosThreshold(threshold),
		z.WithAntiEntropy(time.Second), z.WithDisabledPKI())
	defer node2.Stop()

	// Note: we set the heartbeat and antientropy so that node3 will annonce
	// itself and get rumors.
	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
		z.WithTotalPeers(3),
		z.WithPaxosID(3),
		z.WithPaxosThreshold(threshold),
		z.WithHeartbeat(time.Hour),
		z.WithAntiEntropy(time.Second), z.WithDisabledPKI())
	defer node3.Stop()

	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node1.GetAddr())

	tagDone := make(chan struct{})

	go func() {
		err := node1.SetUsername("A")
		require.NoError(t, err)

		close(tagDone)
	}()

	time.Sleep(time.Second * 3)

	select {
	case <-tagDone:
		t.Error(t, "a consensus can't be reached")
	default:
	}

	// > Add a new peer: with 3 peers a consensus can now be reached. Node3 has
	// the heartbeat so it will annonce itself to node1.
	node3.AddPeer(node1.GetAddr())

	timeout := time.After(time.Second * 10)

	select {
	case <-tagDone:
	case <-timeout:
		t.Error(t, "a consensus must have been reached")
	}

	// wait for rumors to be spread, especially TLC messages.
	time.Sleep(time.Second * 3)

	// > node1 name store is updated

	names := node1.GetStorage().GetNamingStore()
	require.Equal(t, 1, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get("A"))

	// > node2 name store is updated

	names = node2.GetStorage().GetNamingStore()
	require.Equal(t, 1, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get("A"))

	// > node3 name store is updated

	names = node3.GetStorage().GetNamingStore()
	require.Equal(t, 1, names.Len())
	require.Equal(t, []byte(node1.GetAddr()), names.Get("A"))

	time.Sleep(time.Second)

	// > all nodes must have broadcasted 1 TLC message. There could be more sent
	// if the node replied to a status from a peer that missed the broadcast.

	tlcMsgs := getTLCMessagesFromRumors(t, node1.GetOuts(), node1.GetAddr())
	require.GreaterOrEqual(t, len(tlcMsgs), 1)

	tlcMsgs = getTLCMessagesFromRumors(t, node2.GetOuts(), node2.GetAddr())
	require.GreaterOrEqual(t, len(tlcMsgs), 1)

	tlcMsgs = getTLCMessagesFromRumors(t, node3.GetOuts(), node3.GetAddr())
	require.GreaterOrEqual(t, len(tlcMsgs), 1)
}

// Test if the node can catch up TLC messages
// First message: {"OldName":"","NewName":"a"},
// Second message: b,
// Third message: {"OldName":"a","NewName":"c"}
// There will be in total 2 records in the naming store.
// This is because the third message will override the naming store record, as they come from the same node.
func Test_Username_TLC_Catchup(t *testing.T) {
	transp := channel.NewTransport()

	// Threshold = 1
	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithAckTimeout(0), z.WithTotalPeers(1), z.WithDisabledPKI())
	defer node1.Stop()

	socketX, err := transp.CreateSocket("127.0.0.1:0")
	require.NoError(t, err)

	node1.AddPeer(socketX.GetAddress())

	// send for step 2

	blockHash := make([]byte, 32)
	blockHash[0] = 3

	previousHash := make([]byte, 32)
	previousHash[0] = 2

	tlc := types.TLCMessage{
		Step: 2,
		Block: types.BlockchainBlock{
			Index: 0,
			Hash:  blockHash,
			Value: types.PaxosValue{
				UniqID:   "xxx",
				Filename: `{"OldName":"a","NewName":"c"}`,
				Metahash: "1",
			},
			PrevHash: previousHash,
		},
	}

	transpMsg, err := node1.GetRegistry().MarshalMessage(&tlc)
	require.NoError(t, err)

	header := transport.NewHeader(socketX.GetAddress(), socketX.GetAddress(), node1.GetAddr(), 0)

	packet := transport.Packet{
		Header: &header,
		Msg:    &transpMsg,
	}

	err = socketX.Send(node1.GetAddr(), packet, 0)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	// Send for step 1

	tlc.Step = 1

	blockHash[0] = 2
	previousHash[0] = 1

	tlc.Block.Hash = blockHash
	tlc.Block.PrevHash = previousHash

	tlc.Block.Value.Filename = "b"
	tlc.Block.Value.Metahash = "2"

	transpMsg, err = node1.GetRegistry().MarshalMessage(&tlc)
	require.NoError(t, err)

	header = transport.NewHeader(socketX.GetAddress(), socketX.GetAddress(), node1.GetAddr(), 0)

	packet = transport.Packet{
		Header: &header,
		Msg:    &transpMsg,
	}

	err = socketX.Send(node1.GetAddr(), packet, 0)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	// > at this stage no blocks are added

	store := node1.GetStorage().GetBlockchainStore()
	require.Equal(t, 0, store.Len())

	// adding the expected TLC message. Peer must then add block 0, 1, and 2.

	tlc.Step = 0

	blockHash[0] = 1
	previousHash[0] = 0

	tlc.Block.Hash = blockHash
	tlc.Block.PrevHash = previousHash

	tlc.Block.Value.Filename = `{"OldName":"","NewName":"a"}`
	tlc.Block.Value.Metahash = "1"

	transpMsg, err = node1.GetRegistry().MarshalMessage(&tlc)
	require.NoError(t, err)

	header = transport.NewHeader(socketX.GetAddress(), socketX.GetAddress(), node1.GetAddr(), 0)

	packet = transport.Packet{
		Header: &header,
		Msg:    &transpMsg,
	}

	err = socketX.Send(node1.GetAddr(), packet, 0)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	// > node1 must have 3 blocks in its block store

	// 3 blocks + the last block key
	require.Equal(t, 4, store.Len())
	blockBuf := store.Get(hex.EncodeToString(blockHash))

	var block types.BlockchainBlock
	err = block.Unmarshal(blockBuf)
	require.NoError(t, err)

	require.Equal(t, tlc.Block, block)

	// > node1 must have the block hash in the LasBlockKey store

	blockHash[0] = 3
	require.Equal(t, blockHash, store.Get(storage.LastBlockKey))

	// > node1 must have 3 names in its name store

	require.Equal(t, 2, node1.GetStorage().GetNamingStore().Len())
	require.Equal(t, []byte("1"), node1.GetStorage().GetNamingStore().Get("c"))
	require.Equal(t, []byte("2"), node1.GetStorage().GetNamingStore().Get("b"))
}

// node1 and node2 set username 5 times
// Check if node3 can catch up
// In total there will be two records and 11 blocks in each node.
func Test_Username_SetUsername_Catchup(t *testing.T) {
	transp := channel.NewTransport()

	numBlocks := 5

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(3), z.WithPaxosID(1), z.WithDisabledPKI())
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(3), z.WithPaxosID(2), z.WithDisabledPKI())
	defer node2.Stop()

	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node1.GetAddr())

	// Setting N blocks

	for i := 0; i < numBlocks; i++ {
		err := node1.SetUsername(strconv.Itoa(i))
		require.NoError(t, err)
	}

	for i := 0; i < numBlocks; i++ {
		err := node2.SetUsername(strconv.Itoa(numBlocks + i))
		require.NoError(t, err)
	}

	time.Sleep(time.Second * 2)

	// > at this stage node1 and node2 must have 11 blocks in their blockchain
	// store and 2 names in their name store.

	require.Equal(t, 2, node1.GetStorage().GetNamingStore().Len())
	require.Equal(t, 2, node2.GetStorage().GetNamingStore().Len())

	require.Equal(t, 11, node1.GetStorage().GetBlockchainStore().Len())
	require.Equal(t, 11, node2.GetStorage().GetBlockchainStore().Len())

	// > let's add the third peer and see if it can catchup.

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(3), z.WithPaxosID(3), z.WithDisabledPKI())
	defer node3.Stop()

	node3.AddPeer(node2.GetAddr())

	msg := types.EmptyMessage{}

	transpMsg, err := node3.GetRegistry().MarshalMessage(msg)
	require.NoError(t, err)

	// > by broadcasting a message node3 will get back an ack with a status,
	// making it asking for the missing rumors.

	err = node3.Broadcast(transpMsg)
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	// > checking the name and blockchain stores

	require.Equal(t, 2, node3.GetStorage().GetNamingStore().Len())
	require.Equal(t, 11, node3.GetStorage().GetBlockchainStore().Len())

	// > check that all blockchain store have the same last block hash

	blockStore1 := node1.GetStorage().GetBlockchainStore()
	blockStore2 := node2.GetStorage().GetBlockchainStore()
	blockStore3 := node3.GetStorage().GetBlockchainStore()

	require.Equal(t, blockStore1.Get(storage.LastBlockKey), blockStore2.Get(storage.LastBlockKey))
	require.Equal(t, blockStore1.Get(storage.LastBlockKey), blockStore3.Get(storage.LastBlockKey))

	// > validate the chain in each store

	z.ValidateBlockchain(t, blockStore1)
	z.ValidateBlockchain(t, blockStore2)
	z.ValidateBlockchain(t, blockStore3)
}

// 3 nodes concurrently modify its name 7 times.
// After the operation, there will be 3 naming store records in each node
func Test_Username_SetUsername_Stress_Test_And_Search(t *testing.T) {
	numMessages := 7
	numNodes := 3

	transp := channel.NewTransport()

	nodes := make([]z.TestNode, numNodes)

	for i := range nodes {
		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(uint(numNodes)), z.WithPaxosID(uint(i+1)), z.WithDisabledPKI())
		defer node.Stop()

		nodes[i] = node
	}

	for _, n1 := range nodes {
		for _, n2 := range nodes {
			n1.AddPeer(n2.GetAddr())
		}
	}

	wait := sync.WaitGroup{}
	wait.Add(numNodes)

	patternString := "ThisIsAPrefix-"

	for _, node := range nodes {
		n := node
		go func(n z.TestNode) {
			defer wait.Done()

			for i := 0; i < numMessages; i++ {
				prefix := fmt.Sprintf("%v-%v-%v-", patternString, i, n.GetAddr())

				err := n.SetUsername(prefix + randomString(10))
				require.NoError(t, err)

				time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))
			}
		}(n)
	}

	wait.Wait()

	time.Sleep(time.Second * 2)

	lastHashes := map[string]struct{}{}

	for i, node := range nodes {
		t.Logf("node %d", i)

		store := node.GetStorage().GetBlockchainStore()
		require.Equal(t, numMessages*numNodes+1, store.Len())

		lastHashes[string(store.Get(storage.LastBlockKey))] = struct{}{}

		z.ValidateBlockchain(t, store)

		require.Equal(t, numNodes, node.GetStorage().GetNamingStore().Len())

		z.DisplayLastBlockchainBlock(t, os.Stdout, node.GetStorage().GetBlockchainStore())
	}

	// > all peers must have the same last hash
	require.Len(t, lastHashes, 1)

	for i, node := range nodes {
		t.Logf("node %d", i)
		results, err := node.SearchUsername(patternString + "-")
		require.NoError(t, err)
		require.Len(t, results, numNodes)
	}
}

// Helper methods
func randomString(n int) string {
	letters := []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ{},:")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// -----------------------------------------------------------------------------
// Utility functions

// getRumor returns the transport.Message embedded in the rumor at the provided
// sequence.
func getRumor(t *testing.T, pkts []transport.Packet, sequence uint) (*transport.Message, *transport.Header) {
	for _, pkt := range pkts {
		if pkt.Msg.Type == "rumor" {
			rumor := z.GetRumor(t, pkt.Msg)

			// a broadcast only have one rumor
			if len(rumor.Rumors) == 1 {
				if rumor.Rumors[0].Sequence == sequence {
					return rumor.Rumors[0].Msg, pkt.Header
				}
			}
		}
	}
	return nil, nil
}

// getTLCMessagesFromRumors returns the TLC messages from rumor messages. We're
// expecting the rumor message to contain only one rumor that embeds the TLC
// message. The rumor originates from the given addr.
func getTLCMessagesFromRumors(t *testing.T, outs []transport.Packet, addr string) []types.TLCMessage {
	var result []types.TLCMessage

	for _, msg := range outs {
		if msg.Msg.Type == "rumor" {
			rumor := z.GetRumor(t, msg.Msg)
			if len(rumor.Rumors) == 1 && rumor.Rumors[0].Msg.Type == "tlc" {
				if rumor.Rumors[0].Origin == addr {
					tlc := z.GetTLC(t, rumor.Rumors[0].Msg)
					result = append(result, tlc)
				}
			}
		}
	}

	return result
}
