package unit

import (
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/transport/channel"
	"go.dedis.ch/cs438/types"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

// Simple scenario
// node1 -> id=1
// node2 -> id=2
// node1 tries to elect the leader
// Messages:
// node1 -> node2 Election
// node2 -> node1 Answer
// node2 -> node1 Coordinator
// node2 -> node2 Coordinator
func Test_Bully_Simple_Select_Correct_Coordinator(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(2), z.WithPaxosID(1), z.WithDisabledPKI())
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(2), z.WithPaxosID(2), z.WithDisabledPKI())
	defer node2.Stop()

	bullyMap := make(map[uint]string)
	bullyMap[1] = node1.GetAddr()
	bullyMap[2] = node2.GetAddr()

	node1.ConfigurePeers(bullyMap)
	node2.ConfigurePeers(bullyMap)

	node1.AddPeer(node2.GetAddr())
	node1.Elect()

	time.Sleep(time.Second * 4)

	n1outs := node1.GetOuts()
	n2outs := node2.GetOuts()

	require.Len(t, n1outs, 1)
	require.Len(t, n2outs, 3)

	n1Bully := z.GetBully(t, n1outs[0].Msg)
	require.Equal(t, types.BullyElection, n1Bully.Type)

	n2Bully := z.GetBully(t, n2outs[0].Msg)
	require.Equal(t, types.BullyAnswer, n2Bully.Type)

	n2Bully = z.GetBully(t, n2outs[1].Msg)
	require.Equal(t, types.BullyCoordinator, n2Bully.Type)

	n2Bully = z.GetBully(t, n2outs[2].Msg)
	require.Equal(t, types.BullyCoordinator, n2Bully.Type)

	// now both node1 and node2 should have the same leader
	require.Equal(t, uint(2), node2.GetCoordinator())
	require.Equal(t, uint(2), node1.GetCoordinator())

}

// The test extends the above test with a complex topology
// The node with the largest PaxosID should become the leader eventually.
func Test_Bully_Complex_Select_Correct_Coordinator(t *testing.T) {
	transp := channel.NewTransport()

	numNodes := 5
	nodes := make([]z.TestNode, numNodes)
	bullyMap := make(map[uint]string)
	for i := 1; i <= numNodes; i++ {
		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(uint(numNodes)), z.WithPaxosID(uint(i)), z.WithDisabledPKI())
		defer node.Stop()
		bullyMap[uint(i)] = node.GetAddr()
		nodes[i-1] = node
	}

	for _, v := range nodes {
		v.ConfigurePeers(bullyMap)
	}

	nodes[0].Elect()
	time.Sleep(time.Second * 10)

	for _, v := range nodes {
		require.Equal(t, uint(numNodes), v.GetCoordinator())
	}

}

// The test performs both leader election and set username with a simple topology
func Test_Bully_Simple_Select_Correct_Coordinator_And_SetUsername(t *testing.T) {
	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(2), z.WithPaxosID(1), z.WithDisabledPKI())
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(2), z.WithPaxosID(2), z.WithDisabledPKI())
	defer node2.Stop()

	bullyMap := make(map[uint]string)
	bullyMap[1] = node1.GetAddr()
	bullyMap[2] = node2.GetAddr()

	node1.ConfigurePeers(bullyMap)
	node2.ConfigurePeers(bullyMap)

	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node1.GetAddr())
	node1.Elect()

	time.Sleep(time.Second * 4)

	n1outs := node1.GetOuts()
	n2outs := node2.GetOuts()

	require.Len(t, n1outs, 1)
	require.Len(t, n2outs, 3)

	n1Bully := z.GetBully(t, n1outs[0].Msg)
	require.Equal(t, types.BullyElection, n1Bully.Type)

	n2Bully := z.GetBully(t, n2outs[0].Msg)
	require.Equal(t, types.BullyAnswer, n2Bully.Type)

	n2Bully = z.GetBully(t, n2outs[1].Msg)
	require.Equal(t, types.BullyCoordinator, n2Bully.Type)

	n2Bully = z.GetBully(t, n2outs[2].Msg)
	require.Equal(t, types.BullyCoordinator, n2Bully.Type)

	// now both node1 and node2 should have the same leader
	require.Equal(t, uint(2), node2.GetCoordinator())
	require.Equal(t, uint(2), node1.GetCoordinator())

	node1NewName := "this is my new fancy name!"
	jsonValue, _ := peer.BuildUsernamePaxosValue(node1NewName, "")

	node1.SetUsernameWithCoordinator(node1NewName)
	time.Sleep(time.Second * 2)

	// > node1 blockchain store contains two elements
	bstore := node1.GetStorage().GetBlockchainStore()
	require.Equal(t, 2, bstore.Len())
	lastBlockHash := bstore.Get(storage.LastBlockKey)
	lastBlock := bstore.Get(hex.EncodeToString(lastBlockHash))
	var block types.BlockchainBlock
	err := block.Unmarshal(lastBlock)
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
	err = node2.SetUsernameWithCoordinator(node1NewName)
	require.Error(t, err)

	// Now if node1 tries to set it again, there will be no error
	err = node1.SetUsernameWithCoordinator(node1NewName)
	require.NoError(t, err)

	//////////////////////////////////////////////////////////////////////////
	// Now let's set node2 to another name
	node2NewName := "this is the new name for node 2"
	err = node2.SetUsernameWithCoordinator(node2NewName)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > node1 name store is updated
	names := node1.GetStorage().GetNamingStore()
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
	err = node1.SetUsernameWithCoordinator(node1NewNewName)
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
	err = node2.SetUsernameWithCoordinator(node1NewName)
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

// The test performs both leader election and set username with a complex topology
func Test_Bully_SetUsernameWithCoordinator_Stress_Test_And_Search(t *testing.T) {
	numMessages := 3
	numNodes := 3

	transp := channel.NewTransport()

	nodes := make([]z.TestNode, numNodes)
	bullyMap := make(map[uint]string)

	for i := range nodes {
		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(uint(numNodes)), z.WithPaxosID(uint(i+1)), z.WithDisabledPKI())
		defer node.Stop()
		bullyMap[uint(i+1)] = node.GetAddr()
		nodes[i] = node
	}

	for _, n := range nodes {
		n.ConfigurePeers(bullyMap)
	}

	for _, n1 := range nodes {
		for _, n2 := range nodes {
			n1.AddPeer(n2.GetAddr())
		}
	}

	nodes[0].Elect()
	time.Sleep(time.Second * 10)

	for _, v := range nodes {
		require.Equal(t, uint(numNodes), v.GetCoordinator())
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

				err := n.SetUsernameWithCoordinator(prefix)
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

		z.DisplayLastBlockchainBlock(t, os.Stdout, node.GetStorage().GetBlockchainStore())
	}

	// > all peers must have the same last hash
	require.Len(t, lastHashes, 1)

	for i, node := range nodes {
		t.Logf("node %d", i)
		prefix := fmt.Sprintf("%v-%v-", patternString, numMessages-1)
		_, err := node.SearchUsername(prefix)
		require.NoError(t, err)
	}
}
