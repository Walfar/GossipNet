package types

import (
	"crypto/rsa"
	"go.dedis.ch/cs438/transport"
	"time"
)

// ChatMessage is a message sent to exchange text messages between nodes.
//
// - implements types.Message
// - implemented in HW0
type ChatMessage struct {
	Message string
}

// RumorsMessage is a type of message that uses gossip mechanisms to ensure
// reliable delivery. It will eventually be distributed over all nodes.
//
// - implements types.Message
// - implemented in HW1
type RumorsMessage struct {
	Rumors []Rumor
}

// Rumor wraps a message to ensure delivery to all peers-
type Rumor struct {
	// Origin is the address of the node that initiated the rumor
	Origin string

	// Sequence is the unique ID of the packet from packet's creator point of
	// view. Each time a sender creates a packet, it must increment its sequence
	// number and include it. Start from 1.
	Sequence uint

	// The message the rumor embeds.
	Msg *transport.Message
}

// AckMessage is an acknowledgement message sent back when a node receives a
// rumor. It servers two purpose: (1) tell that it received the message, and (2)
// share its status.
//
// - implements types.Message
// - implemented in HW1
type AckMessage struct {
	// AckedPacketID is the PacketID this acknowledgment is for
	AckedPacketID string
	Status        StatusMessage
}

// StatusMessage describes a status message. It contains the last known sequence
// for an origin. Status messages are used in Ack and by the anti-entropy.
//
// - implements types.Message
// - implemented in HW1
type StatusMessage map[string]uint

// EmptyMessage describes an empty message. It is used for the heartbeat
// mechanism.
//
// - implements types.Message
// - implemented in HW1
type EmptyMessage struct{}

// PrivateMessage describes a message intended to some specific recipients.
//
// - implements types.Message
// - implemented in HW1
type PrivateMessage struct {
	// Recipients is a bag of recipients
	Recipients map[string]struct{}

	// Msg is the private message to be read by the recipients
	Msg *transport.Message
}

type PublicKeyTable map[string]rsa.PublicKey

type PublicKeyMessage struct {
	TablePublicKey PublicKeyTable
}

type NeedConfirmationPKMessage struct {
	AddressPK string
}

type ConfirmationPKMessage struct {
	AddressPK string
	Value rsa.PublicKey
}

type PostProfile struct {
	Message string
	Date    time.Time
}

type RespondProfilePostMessage struct {
	Messages []struct{Message string; Date string}
}

type AskProfilePostMessage struct {
	Address string
}

type NewPost struct {
	Message string
}

type AlertNewPost struct {}
