package types

import "fmt"

// -----------------------------------------------------------------------------
// SetUsernameRequestMessage

// NewEmpty implements types.Message.
func (p SetUsernameRequestMessage) NewEmpty() Message {
	return &SetUsernameRequestMessage{}
}

// Name implements types.Message.
func (p SetUsernameRequestMessage) Name() string {
	return "setusernamerequest"
}

// String implements types.Message.
func (p SetUsernameRequestMessage) String() string {
	return fmt.Sprintf("{setusernamerequest old=%v - new=%v - peer=%v}", p.OldUsername, p.NewUsername, p.PeerAddress)
}

// HTML implements types.Message.
func (p SetUsernameRequestMessage) HTML() string {
	return p.String()
}
