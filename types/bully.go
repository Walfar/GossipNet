package types

import "fmt"

// -----------------------------------------------------------------------------
// BullyMessage

// NewEmpty implements types.Message.
func (p BullyMessage) NewEmpty() Message {
	return &BullyMessage{}
}

// Name implements types.Message.
func (p BullyMessage) Name() string {
	return "bully"
}

// String implements types.Message.
func (p BullyMessage) String() string {
	return fmt.Sprintf("{bully id=%d - type=%d}", p.PaxosID, p.Type)
}

// HTML implements types.Message.
func (p BullyMessage) HTML() string {
	return p.String()
}
