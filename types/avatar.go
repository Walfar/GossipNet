package types

import "fmt"

// -----------------------------------------------------------------------------
// AvatarUpdateMessage

// NewEmpty implements types.Message.
func (p AvatarUpdateMessage) NewEmpty() Message {
	return &AvatarUpdateMessage{}
}

// Name implements types.Message.
func (p AvatarUpdateMessage) Name() string {
	return "avatarupdate"
}

// String implements types.Message.
func (p AvatarUpdateMessage) String() string {
	return fmt.Sprintf("{avatarupdate author=%v- metahash=%v}", p.Author, p.MetahashToDelete)
}

// HTML implements types.Message.
func (p AvatarUpdateMessage) HTML() string {
	return p.String()
}
