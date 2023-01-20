package tenderduty

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

var plainText []byte

func TestEncrypt(t *testing.T) {

	randLen := time.Now().Unix() % 1000 // not really random, just don't want every run to check same size file
	plainText = make([]byte, 997+randLen)
	_, err := rand.Read(plainText)
	if err != nil {
		t.Fatal(err)
	}

	_, err = encrypt(plainText, "password1")
	if err == nil {
		t.Error("well-known passwords should be rejected")
	}
	err = nil
	_, err = encrypt(plainText, "4G*vk90")
	if err == nil {
		t.Error("short passwords should be rejected")
	}
	err = nil

	// try to get some variability in passwords for encrypt/decrypt testing
	passBytes := make([]byte, int(time.Now().Second()%10+9))
	_, err = rand.Read(passBytes)
	if err != nil {
		t.Fatal(err)
	}
	password := make([]byte, base64.StdEncoding.EncodedLen(len(passBytes)))
	base64.StdEncoding.Encode(password, passBytes)

	cipherText, err := encrypt(plainText, string(password[:len(passBytes)]))
	if err != nil {
		t.Error(err)
		return
	}
	if len(cipherText) == 0 {
		t.Error("encryption failed, got 0 length ciphertext")
		return
	}

	plainText2, err := decrypt(cipherText, string(password[:len(passBytes)]))
	if err != nil {
		t.Error("decryption failed", err)
		return
	}
	if len(plainText2) == 0 {
		t.Error("decryption failed, got empty plaintext")
	}
	if !bytes.Equal(plainText, plainText2) {
		t.Error("plaintext does not match after decrypting")
	}

}
