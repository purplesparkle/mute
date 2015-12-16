// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msg

import (
	"bytes"
	"crypto/sha512"
	"io"

	"github.com/mutecomm/mute/cipher"
	"github.com/mutecomm/mute/encode/base64"
	"github.com/mutecomm/mute/msg/session"
	"github.com/mutecomm/mute/util/bzero"
	"golang.org/x/crypto/hkdf"
)

// checkKeys checks that the keys k1, k2, k3, and k4 are pairwise different to
// prevent possible reflection attacks.
func checkKeys(k1, k2, k3, k4 *[32]byte) error {
	if bytes.Equal(k1[:], k2[:]) {
		return ErrReflection
	}
	if bytes.Equal(k1[:], k3[:]) {
		return ErrReflection
	}
	if bytes.Equal(k1[:], k4[:]) {
		return ErrReflection
	}
	if bytes.Equal(k2[:], k3[:]) {
		return ErrReflection
	}
	if bytes.Equal(k2[:], k4[:]) {
		return ErrReflection
	}
	if bytes.Equal(k3[:], k4[:]) {
		return ErrReflection
	}
	return nil
}

// deriveRootKey derives the next root key from t1, t2, t3, and the
// previousRootKeyHash (if it exists).
func deriveRootKey(
	t1, t2, t3 *[32]byte,
	previousRootKeyHash *[64]byte,
) (*[24]byte, error) {
	master := make([]byte, 32+32+32+64)
	copy(master[:], t1[:])
	copy(master[32:], t2[:])
	copy(master[64:], t3[:])
	if previousRootKeyHash != nil {
		copy(master[96:], previousRootKeyHash[:])
	}

	hkdf := hkdf.New(sha512.New, master, nil, nil)

	// derive root key
	// TODO: size correct? Shouldn't it be 64 bytes?
	var rootKey [24]byte
	if _, err := io.ReadFull(hkdf, rootKey[:]); err != nil {
		return nil, err
	}

	return &rootKey, nil
}

// generateMessageKeys generates the next numOfKeys many session keys from
// from rootKey for given senderIdentity and recipientIdentity.
// If recipientKeys is true the generated sender and reciever keys are stored in
// reverse order.
// It uses senderSessionPub and recipientPub in the process and calls
// keyStore.StoresSession and keyStore.SetSessionState to store the result.
func generateMessageKeys(
	senderIdentity, recipientIdentity string,
	rootKey *[24]byte,
	recipientKeys bool,
	senderSessionPub, recipientPub *[32]byte,
	numOfKeys int,
	keyStore session.Store,
) error {
	var (
		identities string
		send       []string
		recv       []string
	)

	// identity_fix = HASH(SORT(SenderNym, RecipientNym))
	if senderIdentity < recipientIdentity {
		identities = senderIdentity + recipientIdentity
	} else {
		identities = recipientIdentity + senderIdentity
	}
	identityFix := cipher.SHA512([]byte(identities))
	recipientPubHash := cipher.SHA512(recipientPub[:])
	senderSessionPubHash := cipher.SHA512(senderSessionPub[:])

	chainKey := rootKey[:]
	for i := 0; i < numOfKeys; i++ {
		// messagekey_send[i] = HMAC_HASH(chainkey, "MESSAGE" | HASH(RecipientPub) | identity_fix)
		buffer := append([]byte("MESSAGE"), recipientPubHash...)
		buffer = append(buffer, identityFix...)
		send = append(send, base64.Encode(cipher.HMAC(chainKey, buffer)))

		// messagekey_recv[i] = HMAC_HASH(chainkey, "MESSAGE" | HASH(SenderSessionPub) | identity_fix)
		buffer = append([]byte("MESSAGE"), senderSessionPubHash...)
		buffer = append(buffer, identityFix...)
		recv = append(recv, base64.Encode(cipher.HMAC(chainKey, buffer)))

		// chainkey = HMAC_HASH(chainkey, "CHAIN" )
		chainKey = cipher.HMAC(chainKey, []byte("CHAIN"))
	}

	// calculate root key hash
	rootKeyHash := base64.Encode(cipher.SHA512(rootKey[:]))
	bzero.Bytes(rootKey[:])

	// reverse key material, if necessary
	if recipientKeys {
		senderIdentity, recipientIdentity = recipientIdentity, senderIdentity
		send, recv = recv, send
	}

	// store session
	var pubHash string
	if recipientKeys {
		pubHash = base64.Encode(cipher.SHA512(recipientPub[:]))
	} else {
		pubHash = base64.Encode(cipher.SHA512(senderSessionPub[:]))
	}
	err := keyStore.StoreSession(senderIdentity, recipientIdentity, pubHash,
		rootKeyHash, base64.Encode(chainKey), send, recv)
	if err != nil {
		return err
	}

	return nil
}

// deriveSymmetricKeys derives the symmetric cryptoKey and hmacKey from the
// given messageKey.
func deriveSymmetricKeys(messageKey *[64]byte) (
	cryptoKey, hmacKey []byte,
	err error,
) {
	// TODO: set optional salt and info?
	hkdf := hkdf.New(sha512.New, messageKey[:], nil, nil)

	// derive crypto key for AES-256
	cryptoKey = make([]byte, 32)
	if _, err := io.ReadFull(hkdf, cryptoKey); err != nil {
		return nil, nil, err
	}

	// derive HMAC key for SHA-512 HMAC (TODO: correct size?)
	hmacKey = make([]byte, 64)
	if _, err := io.ReadFull(hkdf, hmacKey); err != nil {
		return nil, nil, err
	}

	return
}
