package types

import (
	"go.dedis.ch/cs438/transport"
)

// AddPeerArgument is the json type to call messaging.AddPeer()
type AddPeerArgument []string

// UnicastArgument is the json type to call messaging.Unicast()
type UnicastArgument struct {
	Dest string
	Msg  transport.Message
}

// SetRoutingEntryArgument is the json type to call messaging.SetRoutingEntry()
type SetRoutingEntryArgument struct {
	Origin    string
	RelayAddr string
}

// IndexArgument is the json type to call datasharing.SearchAndIndex()
type IndexArgument struct {
	Pattern string
	Budget  uint
	Timeout string
}

// SearchArgument is the json type to call datasharing.SearchFirst()
type SearchArgument struct {
	Pattern string

	Initial uint
	Factor  uint
	Retry   uint
	Timeout string
}

type SendFriendRequestArgument struct {
	Dest string
}

type EncryptedMessageArgument struct {
	Dest string
	Msg  string
}

type SendPositiveResponseArgument struct {
	Dest string
}

type SendNegativeResponseArgument struct {
	Dest string
}

type SetPersonalPost struct {
	Message string
}

type AskPersonalPost struct {
	Dest string
	Msg transport.Message
}