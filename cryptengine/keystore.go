// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cryptengine

import (
	"database/sql"

	"github.com/mutecomm/mute/encode/base64"
	"github.com/mutecomm/mute/log"
	"github.com/mutecomm/mute/msg/session"
	"github.com/mutecomm/mute/uid"
	"github.com/mutecomm/mute/util"
)

// GetSessionState implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetSessionState(sessionStateKey string) (
	*session.State,
	error,
) {
	ss, err := ce.keyDB.GetSessionState(sessionStateKey)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

// SetSessionState implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) SetSessionState(
	sessionStateKey string,
	sessionState *session.State,
) error {
	return ce.keyDB.SetSessionState(sessionStateKey, sessionState)
}

// StoreSession implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) StoreSession(
	sessionKey, rootKeyHash, chainKey string,
	send, recv []string,
) error {
	return ce.keyDB.AddSession(sessionKey, rootKeyHash, chainKey, send, recv)
}

// HasSession implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) HasSession(sessionKey string) bool {
	_, _, _, err := ce.keyDB.GetSession(sessionKey)
	switch {
	case err == sql.ErrNoRows:
		return false
	case err != nil:
		// TODO: handle this without panic!
		panic(log.Critical(err))
	}
	return true
}

// GetPrivateKeyEntry implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetPrivateKeyEntry(pubKeyHash string) (*uid.KeyEntry, error) {
	log.Debugf("ce.FindKeyEntry: pubKeyHash=%s", pubKeyHash)
	ki, sigPubKey, privateKey, err := ce.keyDB.GetPrivateKeyInit(pubKeyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, session.ErrNoKeyEntry
		}
		return nil, err
	}
	// decrypt KeyEntry
	ke, err := ki.KeyEntryECDHE25519(sigPubKey)
	if err != nil {
		return nil, err
	}
	// set private key
	if err := ke.SetPrivateKey(privateKey); err != nil {
		return nil, err
	}
	return ke, nil
}

// GetPublicKeyEntry implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetPublicKeyEntry(uidMsg *uid.Message) (*uid.KeyEntry, string, error) {
	log.Debugf("ce.FindKeyEntry: uidMsg.Identity()=%s", uidMsg.Identity())
	// get KeyInit
	sigKeyHash, err := uidMsg.SigKeyHash()
	if err != nil {
		return nil, "", err
	}
	ki, err := ce.keyDB.GetPublicKeyInit(sigKeyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", session.ErrNoKeyEntry
		}
		return nil, "", err
	}
	// decrypt SessionAnchor
	sa, err := ki.SessionAnchor(uidMsg.SigPubKey())
	if err != nil {
		return nil, "", err
	}
	// get KeyEntry message from SessionAnchor
	ke, err := sa.KeyEntry("ECDHE25519")
	if err != nil {
		return nil, "", err
	}
	return ke, sa.NymAddress(), nil
}

// GetMessageKey implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetMessageKey(
	sessionKey string,
	sender bool,
	msgIndex uint64,
) (*[64]byte, error) {
	key, err := ce.keyDB.GetMessageKey(sessionKey, sender, msgIndex)
	if err != nil {
		return nil, err
	}
	// decode key
	var messageKey [64]byte
	k, err := base64.Decode(key)
	if err != nil {
		return nil,
			log.Errorf("cryptengine: cannot decode key for %s", sessionKey)
	}
	if copy(messageKey[:], k) != 64 {
		return nil,
			log.Errorf("cryptengine: key for %s has wrong length", sessionKey)
	}
	return &messageKey, nil
}

// NumMessageKeys implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) NumMessageKeys(sessionKey string) (uint64, error) {
	_, _, n, err := ce.keyDB.GetSession(sessionKey)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// GetRootKeyHash implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetRootKeyHash(sessionKey string) (*[64]byte, error) {
	rootKeyHash, _, _, err := ce.keyDB.GetSession(sessionKey)
	if err != nil {
		return nil, err
	}
	// decode root key hash
	var hash [64]byte
	k, err := base64.Decode(rootKeyHash)
	if err != nil {
		return nil, log.Error("cryptengine: cannot decode root key hash")
	}
	if copy(hash[:], k) != 64 {
		return nil, log.Errorf("cryptengine: root key hash has wrong length")
	}
	return &hash, nil
}

// GetChainKey implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetChainKey(sessionKey string) (*[32]byte, error) {
	_, chainKey, _, err := ce.keyDB.GetSession(sessionKey)
	if err != nil {
		return nil, err
	}
	// decode chain key
	var key [32]byte
	k, err := base64.Decode(chainKey)
	if err != nil {
		return nil, log.Error("cryptengine: cannot decode chain key")
	}
	if copy(key[:], k) != 32 {
		return nil, log.Errorf("cryptengine: chain key has wrong length")
	}
	return &key, nil
}

// DelMessageKey implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) DelMessageKey(
	sessionKey string,
	sender bool,
	msgIndex uint64,
) error {
	return ce.keyDB.DelMessageKey(sessionKey, sender, msgIndex)
}

// AddSessionKey implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) AddSessionKey(
	hash, json, privKey string,
	cleanupTime uint64,
) error {
	return ce.keyDB.AddSessionKey(hash, json, privKey, cleanupTime)
}

// GetSessionKey implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) GetSessionKey(hash string) (
	json, privKey string,
	err error,
) {
	json, privKey, err = ce.keyDB.GetSessionKey(hash)
	switch {
	case err == sql.ErrNoRows:
		return "", "", log.Error(session.ErrNoKeyEntry)
	case err != nil:
		return "", "", err
	}
	return
}

// DelPrivSessionKey implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) DelPrivSessionKey(hash string) error {
	return ce.keyDB.DelPrivSessionKey(hash)
}

// CleanupSessionKeys implements corresponding method for msg.KeyStore interface.
func (ce *CryptEngine) CleanupSessionKeys(t uint64) error {
	return util.ErrNotImplemented
}
