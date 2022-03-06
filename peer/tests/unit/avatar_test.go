package unit

import (
	"bufio"
	"bytes"
	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport/channel"
	"testing"
	"time"
)

// Consider the topology:
// 	node1 <-> node2 <-> node3
// We upload and replace avatar at node1 and node3
// All nodes in the graph should be able to download the avatar of node1 and node3
func Test_Avatar_Upload_Download(t *testing.T) {
	transp := channel.NewTransport()

	var avatar []byte

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithDisabledPKI(), z.WithTotalPeers(3), z.WithPaxosID(1), z.WithChunkSize(3))
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithDisabledPKI(), z.WithTotalPeers(3), z.WithPaxosID(2), z.WithChunkSize(3))
	defer node2.Stop()

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithDisabledPKI(), z.WithTotalPeers(3), z.WithPaxosID(3), z.WithChunkSize(3))
	defer node3.Stop()

	node1.AddPeer(node2.GetAddr())
	node2.AddPeer(node1.GetAddr())

	node2.AddPeer(node3.GetAddr())
	node3.AddPeer(node2.GetAddr())

	chunks := [][]byte{{'a', 'a', 'a'}, {'b', 'b', 'b'}}
	data := append(chunks[0], chunks[1]...)

	// sha256 of each chunk, computed by hand
	chunkHashes := []string{
		"9834876dcfb05cb167a5c24953eba58c4ac89b1adf57f28f2f9d09af107ee8f0",
		"3e744b9dc39389baf0c5a0660589b8402f3dbb49b89b3e75f2c9355852a3c677",
	}

	// metahash, computed by hand
	mh := "6a0b1d67884e58786e97bc51544cbba4cc3e1279d8ff46da2fa32bcdb44a053e"

	buf := bytes.NewBuffer(data)

	_, errorBefore := node1.DownloadAvatar(node1.GetAddr())
	require.Error(t, errorBefore)

	// > we should be able to store data without error
	metaHash, err := node1.UploadAvatar(buf)
	require.NoError(t, err)
	time.Sleep(time.Second * 2)

	// > returned metahash should be the expected one
	require.Equal(t, mh, metaHash)

	store := node1.GetStorage().GetDataBlobStore()

	// > val should contain 3 lines, one for each hex encoded hash of chunk
	val := store.Get(mh)
	scanner := bufio.NewScanner(bytes.NewBuffer(val))

	for i, chunkHash := range chunkHashes {
		ok := scanner.Scan()
		require.True(t, ok)

		require.Equal(t, chunkHash, scanner.Text())

		chunkData := store.Get(scanner.Text())
		require.Equal(t, chunks[i], chunkData)
	}

	// > there should be no additional lines
	ok := scanner.Scan()
	require.False(t, ok)

	require.Equal(t, 1, node1.GetStorage().GetNamingStore().Len())
	require.Equal(t, 1, node2.GetStorage().GetNamingStore().Len())
	require.Equal(t, 1, node3.GetStorage().GetNamingStore().Len())

	_, err = node1.DownloadAvatar(node2.GetAddr())
	require.Error(t, err)

	_, err = node3.DownloadAvatar(node2.GetAddr())
	require.Error(t, err)

	avatar, err = node3.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	_, err = node1.DownloadAvatar(node3.GetAddr())
	require.Error(t, err)

	avatar, err = node2.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	_, err = node2.DownloadAvatar(node3.GetAddr())
	require.Error(t, err)

	avatar, err = node1.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	/////////////////////////////////////////
	// Now lets update node1's avatar
	chunks = [][]byte{{'a', 'a', 'a'}, {'b', 'b', 'b'}, {'c'}}
	data = append(chunks[0], append(chunks[1], chunks[2]...)...)

	// sha256 of each chunk, computed by hand
	chunkHashes = []string{
		"9834876dcfb05cb167a5c24953eba58c4ac89b1adf57f28f2f9d09af107ee8f0",
		"3e744b9dc39389baf0c5a0660589b8402f3dbb49b89b3e75f2c9355852a3c677",
		"2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6",
	}

	// metahash, computed by hand
	mh = "5a00fc30e073b095a6266136552a3da1d4622d0fdaa057f0b3135aa803321e1c"

	buf = bytes.NewBuffer(data)

	metaHash, err = node1.UploadAvatar(buf)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > returned metahash should be the expected one
	require.Equal(t, mh, metaHash)

	store = node1.GetStorage().GetDataBlobStore()

	// > val should contain 3 lines, one for each hex encoded hash of chunk
	val = store.Get(mh)
	scanner = bufio.NewScanner(bytes.NewBuffer(val))

	for i, chunkHash := range chunkHashes {
		ok := scanner.Scan()
		require.True(t, ok)

		require.Equal(t, chunkHash, scanner.Text())

		chunkData := store.Get(scanner.Text())
		require.Equal(t, chunks[i], chunkData)
	}

	// > there should be no additional lines
	ok = scanner.Scan()
	require.False(t, ok)

	require.Equal(t, 1, node1.GetStorage().GetNamingStore().Len())
	require.Equal(t, 1, node2.GetStorage().GetNamingStore().Len())
	require.Equal(t, 1, node3.GetStorage().GetNamingStore().Len())

	_, err = node1.DownloadAvatar(node2.GetAddr())
	require.Error(t, err)

	_, err = node3.DownloadAvatar(node2.GetAddr())
	require.Error(t, err)

	avatar, err = node3.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	_, err = node1.DownloadAvatar(node3.GetAddr())
	require.Error(t, err)

	avatar, err = node2.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	_, err = node2.DownloadAvatar(node3.GetAddr())
	require.Error(t, err)

	avatar, err = node1.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	/////////////////////////////////////////
	// Now lets update node3
	chunks = [][]byte{{'a', 'a', 'a'}, {'b', 'b', 'b'}, {'c'}}
	data = append(chunks[0], append(chunks[1], chunks[2]...)...)

	// sha256 of each chunk, computed by hand
	chunkHashes = []string{
		"9834876dcfb05cb167a5c24953eba58c4ac89b1adf57f28f2f9d09af107ee8f0",
		"3e744b9dc39389baf0c5a0660589b8402f3dbb49b89b3e75f2c9355852a3c677",
		"2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6",
	}

	// metahash, computed by hand
	mh = "5a00fc30e073b095a6266136552a3da1d4622d0fdaa057f0b3135aa803321e1c"

	buf = bytes.NewBuffer(data)
	metaHash, err = node3.UploadAvatar(buf)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// > returned metahash should be the expected one
	require.Equal(t, mh, metaHash)

	store = node3.GetStorage().GetDataBlobStore()

	// > val should contain 3 lines, one for each hex encoded hash of chunk
	val = store.Get(mh)
	scanner = bufio.NewScanner(bytes.NewBuffer(val))

	for i, chunkHash := range chunkHashes {
		ok := scanner.Scan()
		require.True(t, ok)

		require.Equal(t, chunkHash, scanner.Text())

		chunkData := store.Get(scanner.Text())
		require.Equal(t, chunks[i], chunkData)
	}

	// > there should be no additional lines
	ok = scanner.Scan()
	require.False(t, ok)

	require.Equal(t, 2, node1.GetStorage().GetNamingStore().Len())
	require.Equal(t, 2, node2.GetStorage().GetNamingStore().Len())
	require.Equal(t, 2, node3.GetStorage().GetNamingStore().Len())

	// Now lets download
	// node2 has no avatar
	_, err = node1.DownloadAvatar(node2.GetAddr())
	require.Error(t, err)

	_, err = node3.DownloadAvatar(node2.GetAddr())
	require.Error(t, err)

	avatar, err = node3.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	avatar, err = node1.DownloadAvatar(node3.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	avatar, err = node2.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	avatar, err = node2.DownloadAvatar(node3.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	avatar, err = node1.DownloadAvatar(node1.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	avatar, err = node3.DownloadAvatar(node3.GetAddr())
	require.NoError(t, err)
	require.Equal(t, data, avatar)

	require.Equal(t, 2, node1.GetStorage().GetNamingStore().Len())
	require.Equal(t, 2, node2.GetStorage().GetNamingStore().Len())
	require.Equal(t, 2, node3.GetStorage().GetNamingStore().Len())
}
