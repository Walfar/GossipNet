package impl

import (
	"fmt"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
	"io"
	"regexp"
	"strings"
	"time"
)

func (n *Node) UploadAvatar(data io.Reader) (string, error) {
	metahash, err := n.Upload(data)
	if err == nil {
		oldAvatarHash := n.FindAvatarHash(n.address)

		// Since we use the consensus, this will be available globally
		// Key: userAddress######metahash,
		// Value is the avatar file name for the user
		fullHash := fmt.Sprintf("%v######%v", n.address, metahash)
		err = n.Tag(fullHash, peer.AvatarName(n.address))
		if err != nil {
			return "", err
		}

		if len(oldAvatarHash) > 0 {
			// We need to inform the network to remove any info related to old avatar
			message := types.AvatarUpdateMessage{
				Author:           n.address,
				MetahashToDelete: oldAvatarHash,
			}
			transportMessage, err := n.config.MessageRegistry.MarshalMessage(message)
			if err != nil {
				Logger.Info().Msg(err.Error())
				return "", err
			}
			err = n.Broadcast(transportMessage)
			if err != nil {
				return "", err
			}
		}
	}
	return metahash, err
}

func (n *Node) DownloadAvatar(userAddress string) ([]byte, error) {
	hash := n.FindAvatarHash(userAddress)
	if len(hash) == 0 {
		return nil, xerrors.Errorf("The user does not have an avatar")
	}
	if userAddress != n.address {
		expandingConf := peer.ExpandingRing{
			Initial: 5,
			Factor:  2,
			Retry:   5,
			Timeout: time.Second * 2,
		}
		_, err := n.SearchFirst(*regexp.MustCompile(fmt.Sprintf("%v.*", userAddress)), expandingConf)
		if err != nil {
			return nil, err
		}
	}
	return n.Download(hash)
}

func (n *Node) FindAvatarHash(address string) string {
	metahashHolder := make([]string, 1)
	store := n.config.Storage.GetNamingStore()
	store.ForEach(func(key string, val []byte) bool {
		fileName := string(val)
		if peer.IsAvatarName(fileName) {
			user := strings.ReplaceAll(fileName, "AvatarFile-", "")
			if user == address {
				metahashHolder[0] = key
				return false
			}
		}
		return true
	})
	if len(metahashHolder) <= 0 || len(metahashHolder[0]) <= 0 {
		return ""
	}
	return strings.Split(metahashHolder[0], "######")[1]
}

// ExecAvatarUpdateMessage implements the handler for types.AvatarUpdateMessage
func (n *Node) ExecAvatarUpdateMessage(msg types.Message, pkt transport.Packet) error {
	avatarMessage, ok := msg.(*types.AvatarUpdateMessage)
	if !ok {
		return xerrors.Errorf("Wrong type: %T", msg)
	}
	Logger.Info().Msgf("[%v] ExecAvatarUpdateMessage id=%v: receive AvatarUpdateMessage message: %v", n.address, pkt.Header.PacketID, avatarMessage)

	// Delete the record in the naming store
	fullHash := fmt.Sprintf("%v######%v", avatarMessage.Author, avatarMessage.MetahashToDelete)
	n.config.Storage.GetNamingStore().Delete(fullHash)

	// Delete the record in blob store
	//hashes := n.config.Storage.GetDataBlobStore().Get(avatarMessage.MetahashToDelete)
	//if len(hashes) <= 0 {
	//	return nil
	//}
	//chunkHashes := strings.Split(string(hashes), peer.MetafileSep)
	//for _, chunkHash := range chunkHashes {
	//	// Delete each chunk
	//	n.config.Storage.GetDataBlobStore().Delete(chunkHash)
	//}

	// We only perform a soft delete, i.e. delete the metahash from blob storage, not each chunks
	n.config.Storage.GetDataBlobStore().Delete(avatarMessage.MetahashToDelete)
	return nil
}

func (n *Node) DeleteCatalog(key string) {
	n.catalog.Lock()
	defer n.catalog.Unlock()
	delete(n.catalog.values, key)
}

func (n *Node) GetLocalhost() string {
	return n.address
}
