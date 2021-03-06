// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uid

import (
	"bytes"
	"testing"

	"github.com/mutecomm/mute/cipher"
	"github.com/mutecomm/mute/encode/base64"
	"github.com/mutecomm/mute/keyserver/hashchain"
	"github.com/mutecomm/mute/util/times"
	"golang.org/x/crypto/curve25519"
)

func TestKeyEntry(t *testing.T) {
	// create UID message
	msg, err := Create("alice@mute.berlin", false, "", "", Strict,
		hashchain.TestEntry, cipher.RandReader)
	if err != nil {
		t.Fatal(err)
	}

	// KeyInit
	now := uint64(times.Now())
	ki, _, privateKey, err := msg.KeyInit(1, now+times.Day, now-times.Day,
		false, "mute.berlin", "", "", cipher.RandReader)
	if err != nil {
		t.Fatal(err)
	}

	// KeyEntry
	ke, err := ki.KeyEntryECDHE25519(msg.SigPubKey())
	if err != nil {
		t.Fatal(err)
	}

	// verify consistency
	if err := ke.Verify(); err != nil {
		t.Fatal(err)
	}

	// equal
	if !KeyEntryEqual(ke, ke) {
		t.Error("ke should be equal to itself")
	}
	a := ke
	var b KeyEntry
	b.CIPHERSUITE = a.CIPHERSUITE
	b.FUNCTION = a.FUNCTION
	b.HASH = a.HASH
	b.PUBKEY = a.PUBKEY
	if !KeyEntryEqual(a, &b) {
		t.Error("a and b should be equal to itself")
	}

	// private key check
	if err := ke.SetPrivateKey(privateKey); err != nil {
		t.Fatal(err)
	}
	privKey := ke.PrivateKey32()
	if privateKey != base64.Encode(privKey[:]) {
		t.Error("private keys differ")
	}

	// public key check
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, privKey)
	pubKey := ke.PublicKey32()
	if !bytes.Equal(publicKey[:], pubKey[:]) {
		t.Error("public keys differ")
	}
}

func TestKeyEntryJSON(t *testing.T) {
	var keA KeyEntry
	if err := keA.InitDHKey(cipher.RandReader); err != nil {
		t.Fatal(err)
	}
	pk := keA.PrivateKey()
	jsn := keA.JSON()
	keB, err := NewJSONKeyEntry(jsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := keB.SetPrivateKey(pk); err != nil {
		t.Fatal(err)
	}
	if !KeyEntryEqual(&keA, keB) {
		t.Error("KeyEntry A and B differ")
	}
}
