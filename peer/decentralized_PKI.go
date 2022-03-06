package peer

//package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"

	"go.dedis.ch/cs438/types"
)

//type PublicKeyTable map[string]rsa.PublicKey

type DecentralizedPKI struct {
	privateKey  rsa.PrivateKey
	Table       types.PublicKeyTable
	safeAddress map[string]bool
}

func GenerateKey() (rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048) // here 2048 is the number of bits for RSA
	return *privateKey, err
}

func hashMessage(msg string) ([]byte, error) {
	msgHash := sha256.New()
	_, err := msgHash.Write([]byte(msg))
	msgHashSum := msgHash.Sum(nil)
	return msgHashSum, err
}

func Encrypt(secretMessage string, key rsa.PublicKey) (string, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(secretMessage), label)
	return base64.StdEncoding.EncodeToString(ciphertext), err
}

func (c DecentralizedPKI) Encrypt(secretMessage string, addressPeer string) (string, string, error) {
	if len(secretMessage) == 0 || len(addressPeer) == 0 {
		return "", "", errors.New("message or address equal nil")
	}
	if _, ok := c.Table[addressPeer]; !ok {
		return "", "", errors.New("public key not know for that peer")
	}

	//Encrypt the message
	key := c.Table[addressPeer]
	ciphertext, err := Encrypt(secretMessage, key)
	if err != nil {
		return "", "", errors.New("encryption failed")
	}

	// signature of the message
	msgHash, _ := hashMessage(ciphertext)
	signature, err := rsa.SignPSS(rand.Reader, &c.privateKey, crypto.SHA256, msgHash, nil)
	if err != nil {
		return "", "", errors.New("signature failed")
	}

	signatureString := base64.StdEncoding.EncodeToString(signature)
	return ciphertext, signatureString, err
}

func Decrypt(cipherText string, privKey rsa.PrivateKey) (string, error) {
	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, &privKey, ct, label)
	return string(plaintext), err
}

func (c DecentralizedPKI) Decrypt(cipherText string, signature string, addressPeer string) (string, error) {
	if len(cipherText) == 0 || len(signature) == 0 || len(addressPeer) == 0 {
		return "", errors.New("message or signature or address equal nil")
	}
	if _, ok := c.Table[addressPeer]; !ok {
		return "", errors.New("public key not know for that peer")
	}

	//check the signature
	publicKey := c.Table[addressPeer]
	msgHash, err := hashMessage(cipherText)
	if err != nil {
		return "", err
	}
	signatureByte, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return "", errors.New("signature from string to byte failed")
	}
	err = rsa.VerifyPSS(&publicKey, crypto.SHA256, msgHash, signatureByte, nil)
	if err != nil {
		return "", errors.New("signature failed")
	}

	//Decipher the message
	privKey := c.privateKey
	msg, err := Decrypt(cipherText, privKey)
	if err != nil {
		return "", errors.New("decipher failed")
	}

	return msg, err
}

// initiate
func InitiatePKI(ownAddress string) DecentralizedPKI {

	privateKey, _ := GenerateKey()
	table := make(types.PublicKeyTable)
	table[ownAddress] = privateKey.PublicKey
	safeAddr := make(map[string]bool)
	safeAddr[ownAddress] = true

	c := DecentralizedPKI{
		privateKey:  privateKey,
		Table:       table,
		safeAddress: safeAddr,
	}

	return c
}

func (c DecentralizedPKI) GetPublicKey(addressPeer string) (rsa.PublicKey, bool) {
	val, tableContainKey := c.Table[addressPeer]
	return val, tableContainKey
}

func (c DecentralizedPKI) ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) string {
	pubkey_bytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return ""
	}
	pubkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkey_bytes,
		},
	)

	return string(pubkey_pem)
}

// add publicKey table
func (c DecentralizedPKI) AddKey(key rsa.PublicKey, addressPeer string, neighbor string) (bool, bool, error) {
	needSendCorrection := false
	conflict := false
	oldVal, tableContainKey := c.Table[addressPeer]

	//todo check if it is a safe PK
	_, isSafeAddr := c.safeAddress[addressPeer]
	if isSafeAddr {
		return false, false, nil
	}

	// if address never seen
	if !tableContainKey {
		c.Table[addressPeer] = key
		needSendCorrection = true
		//log.Warn().Msg("---- key never seen")
	} else { // if address already known
		if addressPeer == neighbor { // we know it is the true value because it is from the address

			if c.ExportRsaPublicKeyAsPemStr(&oldVal) != c.ExportRsaPublicKeyAsPemStr(&key) {
				//log.Warn().Msg("---- key seen and it is a neighbor but value is different")
				needSendCorrection = true
			}
			c.Table[addressPeer] = key
			c.safeAddress[addressPeer] = true

		} else { // not sure about the validity of the value

			if c.ExportRsaPublicKeyAsPemStr(&oldVal) != c.ExportRsaPublicKeyAsPemStr(&key) {
				conflict = true
				needSendCorrection = true
				//log.Warn().Msg("---- key seen but value is different")
			}
		}
	}

	return needSendCorrection, conflict, nil
}
