package impl

import (
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
	"regexp"
	"strings"
	"time"
)

func (n *Node) SetUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == peer.EmptyName {
		return xerrors.Errorf("username cannot be empty.")
	}

	n.username.RLock()
	oldName := n.username.value
	n.username.RUnlock()

	if oldName == username {
		Logger.Info().Msgf("[%v] Same username as before, value=%v", n.address, username)
		return nil
	}
	if n.config.Storage.GetNamingStore().Get(username) != nil {
		return xerrors.Errorf("This name: %v is already taken by another user!", username)
	}

	address := n.address
	if n.config.TotalPeers <= 1 {
		if oldName != peer.EmptyName {
			n.config.Storage.GetNamingStore().Delete(oldName)
		}
		n.config.Storage.GetNamingStore().Set(username, []byte(address))
		n.username.Lock()
		n.username.value = username
		n.username.Unlock()
		return nil
	}

	paxosValue, err := peer.BuildUsernamePaxosValue(username, oldName)
	if err != nil {
		return err
	}

	return n.Tag(paxosValue, address)
}

func (n *Node) SetUsernameWithCoordinator(username string) error {
	username = strings.TrimSpace(username)
	if username == peer.EmptyName {
		return xerrors.Errorf("username cannot be empty.")
	}

	n.username.RLock()
	oldName := n.username.value
	n.username.RUnlock()

	if oldName == username {
		Logger.Info().Msgf("[%v] Same username as before, value=%v", n.address, username)
		return nil
	}
	if n.config.Storage.GetNamingStore().Get(username) != nil {
		return xerrors.Errorf("This name: %v is already taken by another user!", username)
	}

	address := n.address
	if n.config.TotalPeers <= 1 {
		if oldName != peer.EmptyName {
			n.config.Storage.GetNamingStore().Delete(oldName)
		}
		n.config.Storage.GetNamingStore().Set(username, []byte(address))
		n.username.Lock()
		n.username.value = username
		n.username.Unlock()
		return nil
	}

	coordinator := n.GetCoordinator()
	if coordinator <= uint(0) {
		return xerrors.Errorf("There is no coordinator!")
	}
	if coordinator == n.config.PaxosID {
		return n.SetUsername(username)
	}

	message := types.SetUsernameRequestMessage{
		OldUsername: oldName,
		NewUsername: username,
		PeerAddress: n.address,
	}
	n.bully.RLock()
	coordinatorAddress := n.bully.network[coordinator]
	n.bully.RUnlock()
	n.sendMessageUnchecked(coordinatorAddress, message)
	return nil
}

func (n *Node) GetUsername() (string, error) {
	n.username.RLock()
	v := n.username.value
	n.username.RUnlock()
	if v == peer.EmptyName {
		return v, xerrors.Errorf("The user does not have a username!")
	}
	return v, nil
}

func (n *Node) SearchUsername(patternStr string) ([]types.SearchUsernameResponse, error) {
	pattern := regexp.MustCompile(patternStr)
	var response []types.SearchUsernameResponse
	n.config.Storage.GetNamingStore().ForEach(
		func(key string, val []byte) bool {
			if strings.Contains(string(val), "AvatarFile-") {
				return true
			}
			if pattern.MatchString(key) {
				match := types.SearchUsernameResponse{
					Username: key,
					Address:  string(val),
				}
				response = append(response, match)
			}
			return true
		})
	if len(response) > 0 {
		return response, nil
	}
	return response, xerrors.Errorf("No match for %v is found", pattern)
}

// ExecSetUsernameRequestMessage implements the handler for types.BullyMessage
func (n *Node) ExecSetUsernameRequestMessage(msg types.Message, pkt transport.Packet) error {
	setUsernameRequestMessage, ok := msg.(*types.SetUsernameRequestMessage)
	if !ok {
		return xerrors.Errorf("Wrong type: %T", msg)
	}
	Logger.Warn().Msgf("[%v] ExecSetUsernameRequestMessage id=%v: receive SetUsernameRequestMessage message: %v", n.address, pkt.Header.PacketID, setUsernameRequestMessage)
	paxosValue, err := peer.BuildUsernamePaxosValue(setUsernameRequestMessage.NewUsername, setUsernameRequestMessage.OldUsername)
	if err != nil {
		return err
	}

	Logger.Warn().Msgf("[%v] I am the coordinator, so I am going to tag = %v", n.address, setUsernameRequestMessage)

	var job = func() {
		err := n.Tag(paxosValue, setUsernameRequestMessage.PeerAddress)
		if err != nil {
			Logger.Warn().Msgf(err.Error())
		}
	}

	select {
	case n.jobQueue <- job:
		break
	case <-time.After(time.Second):
		Logger.Warn().Msgf("[%v] Job queue is full", n.address)
	}

	return nil
}
