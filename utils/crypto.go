package utils

import (
	"encoding/base64"
	"encoding/hex"
	"log/slog"

	cryptoencoding "github.com/cometbft/cometbft/v2/crypto/encoding"
	"github.com/google/uuid"
	"golang.org/x/crypto/sha3"
)

func Sum256(data []byte) string {
	hash := sha3.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func IsPubkeyCorrect(pkType string, pubKey string) bool {

	pubkeyBytes, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return false
	}

	_, err = cryptoencoding.PubKeyFromTypeAndBytes(pkType, pubkeyBytes)
	if err != nil {
		slog.Error("can't decode public key from bytes: %w", err)
		return false
	}

	return true
}

func UUID() string {
	return uuid.New().String()
}
