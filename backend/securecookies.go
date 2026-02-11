package main

import (
	"encoding/base64"
	"fmt"

	"github.com/gorilla/securecookie"
)

// secureCookies provides a means of encoding and decoding secure cookie
// string values, by applying AES encryption and HMAC. If secureCookies
// is empty, encoding and decoding are no-ops, returning the input unchanged.
type secureCookies []securecookie.Codec

func newSecureCookies(encryptionKeys []string) (secureCookies, error) {
	secureCookies := make(secureCookies, len(encryptionKeys))
	for i, encodedKey := range encryptionKeys {
		decodedKey, err := base64.StdEncoding.DecodeString(encodedKey)
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode encryption key: %w", err)
		}
		if n := len(decodedKey); n != 32 && n != 64 {
			return nil, fmt.Errorf("expected encryption key 32 or 64 bytes, got %d", n)
		}
		hashKey := decodedKey
		blockKey := decodedKey[:32] // 32-byte key for AES-256
		sc := securecookie.New(hashKey, blockKey)
		sc.SetSerializer(securecookie.NopEncoder{})
		secureCookies[i] = sc
	}
	return secureCookies, nil
}

func (s secureCookies) Encode(value string) (string, error) {
	if len(s) == 0 {
		return value, nil
	}
	return securecookie.EncodeMulti("", []byte(value), s...)
}

func (s secureCookies) Decode(value string) (string, error) {
	if len(s) == 0 {
		return value, nil
	}
	var out []byte
	err := securecookie.DecodeMulti("", value, &out, s...)
	return string(out), err
}
