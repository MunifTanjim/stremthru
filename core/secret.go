package core

import (
	"errors"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/golang-jwt/jwt/v5"
)

const EncryptionFormat = "AES-GCM-256"

func derive32ByteKey(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}

func Encrypt(secret, value string) (string, error) {
	key := derive32ByteKey(secret)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	nonce_and_ciphertext := aesGCM.Seal(nonce, nonce, []byte(value), nil)

	return util.Base64EncodeByte(nonce_and_ciphertext), nil
}

func Decrypt(secret, value string) (string, error) {
	key := derive32ByteKey(secret)

	nonce_and_ciphertext, err := util.Base64DecodeToByte(value)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := nonce_and_ciphertext[:nonceSize], nonce_and_ciphertext[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

type JWTClaims[T any] struct {
	jwt.RegisteredClaims
	Data *T `json:"data,omitempty"`
}

func CreateJWT[T any](secret string, claims JWTClaims[T]) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseJWT[T any](secretFunc jwt.Keyfunc, encodedToken string, claims *JWTClaims[T], options ...jwt.ParserOption) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(encodedToken, claims, secretFunc, options...)
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if _, ok := token.Claims.(*JWTClaims[T]); ok {
		return token, nil
	}

	return nil, errors.New("malformed token")
}
