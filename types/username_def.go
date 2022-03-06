package types

type SearchUsernameResponse struct {
	Username string
	Address  string
}

type SetUsernameRequestMessage struct {
	OldUsername string
	NewUsername string
	PeerAddress string
}
