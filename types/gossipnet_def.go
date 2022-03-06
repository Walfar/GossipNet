package types

type FriendRequestMessage struct {
}

type PositiveResponse struct {
}

type NegativeResponse struct {
}

type EncryptedMessage struct {
	Ciphertext string
	Signature  string
}
