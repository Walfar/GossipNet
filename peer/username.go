package peer

import (
	"encoding/json"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/types"
)

const EmptyName = ""

type Username interface {
	// SetUsername tries to set up a customized username for the current node
	// If the name is already taken in the global network, an error will be thrown.
	// If the node has already set up a username before, this operation will try to overwrite the old one.
	SetUsername(username string) error

	// GetUsername returns the customized username for the current node
	// If the node does not have a customized username, an error will be thrown.
	GetUsername() (string, error)

	// SearchUsername finds the remote address who owns the name via a regular expression
	SearchUsername(patternStr string) ([]types.SearchUsernameResponse, error)

	// SetUsernameWithCoordinator will use the coordinator to set up names
	// If there is no coordinator, then an error will be thrown
	SetUsernameWithCoordinator(username string) error

	SetLogger(log *zerolog.Logger)
}

type UsernameValue struct {
	OldName string
	NewName string
}

func BuildUsernamePaxosValue(newName string, oldName string) (string, error) {
	v := UsernameValue{
		OldName: oldName,
		NewName: newName,
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ParseUsernamePaxosValue(paxosValue string) (string, string, error) {
	var usernameValue UsernameValue
	err := json.Unmarshal([]byte(paxosValue), &usernameValue)
	if err != nil {
		return "", "", err
	}
	return usernameValue.OldName, usernameValue.NewName, nil
}
