// Copyright (c) 2015 Mute Communications Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cahash

import (
	"encoding/hex"
	"testing"
)

var cacert = "-----BEGIN CERTIFICATE-----\nMIICZTCCAc4CCQDxv0PZsialmTANBgkqhkiG9w0BAQsFADB3MQswCQYDVQQGEwJE\nRTEPMA0GA1UECAwGQmVybGluMQ8wDQYDVQQHDAZCZXJsaW4xETAPBgNVBAoMCE11\ndGUgTHRkMQ8wDQYDVQQLDAZEZXZPcHMxIjAgBgNVBAMMGSouc2VydmljZWd1YXJk\nLmNoYXZwbi5uZXQwHhcNMTUwNzA4MjAwNDI0WhcNMjUwNzA1MjAwNDI0WjB3MQsw\nCQYDVQQGEwJERTEPMA0GA1UECAwGQmVybGluMQ8wDQYDVQQHDAZCZXJsaW4xETAP\nBgNVBAoMCE11dGUgTHRkMQ8wDQYDVQQLDAZEZXZPcHMxIjAgBgNVBAMMGSouc2Vy\ndmljZWd1YXJkLmNoYXZwbi5uZXQwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGB\nALYA93q8vzCm+M8rNCMeRrLbJ2bvT4PRO1g1DKGz8sgEdSW2T9aPYwQEF2tJFsSm\nd2Pp32MBxlb0zkHuu5/XTQD6g6f5FPeZ7lwdrY33mpYp606FXXjX48a7EWu9c3tg\nGKUCZ71cm4UkoBTV1A0Q5A8X0TnwRxGLgNzvpiGBCLVXAgMBAAEwDQYJKoZIhvcN\nAQELBQADgYEAFwPCnrofpwRbLYgrmYbkEH12l29H0bhj5Ljvge+WyyCklT/Ryn7Y\nI2TW0K81xqN5zXvKrrAidOdBydxegdYGSGqBtrpOxumo7Nzyqa5W6Zsue1xUeEdW\nvMBA0Mcaga2CJgOouZAIhArTey5HfCIKFAjUZuTyFoZ80r96kEi9SKY=\n-----END CERTIFICATE-----"
var thash = "04b347294ddc4eb951eb4c3de982151b685f240cd8073782e333aaad60990706337cad942f3a387b26034b366275bc31869aa38469e0774f72de02e6c68e59d5"

func TestHash(t *testing.T) {
	hash, err := Hash([]byte(cacert))
	if err != nil {
		t.Fatalf("CACertHash: %s", err)
	}
	if thash != hex.EncodeToString(hash) {
		t.Error("Hash no match")
	}
}
