package webpush

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateVAPIDKeys will create a private and public VAPID key pair
func (c *Client) GenerateVAPIDKeys() (privateKey, publicKey string, err error) {
	private, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return
	}

	public := private.PublicKey()

	// Convert to base64
	publicKey = base64.RawURLEncoding.EncodeToString(public.Bytes())
	privateKey = base64.RawURLEncoding.EncodeToString(private.Bytes())

	return
}

// Generates the ECDSA public and private keys for the JWT encryption
func generateVAPIDHeaderKeys(privateKey []byte) (*ecdsa.PrivateKey, error) {
	curve := ecdh.P256()

	privKey, err := curve.NewPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	pubKey := privKey.PublicKey()
	pubKeyBytes := pubKey.Bytes()

	if len(pubKeyBytes) != 65 || pubKeyBytes[0] != 0x04 {
		return nil, fmt.Errorf("invalid public key format")
	}

	x := new(big.Int).SetBytes(pubKeyBytes[1:33])
	y := new(big.Int).SetBytes(pubKeyBytes[33:65])

	ecdsaPubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	d := new(big.Int).SetBytes(privateKey)

	return &ecdsa.PrivateKey{
		PublicKey: *ecdsaPubKey,
		D:         d,
	}, nil
}

// getVAPIDAuthorizationHeader
func getVAPIDAuthorizationHeader(
	endpoint,
	subscriber,
	vapidPublicKey,
	vapidPrivateKey string,
	expiration time.Time,
) (string, error) {
	// Create the JWT token
	subURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	// Unless subscriber is an HTTPS URL, assume an e-mail address
	if !strings.HasPrefix(subscriber, "https:") {
		subscriber = "mailto:" + subscriber
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"aud": subURL.Scheme + "://" + subURL.Host,
		"exp": expiration.Unix(),
		"sub": subscriber,
	})

	// Decode the VAPID private key
	decodedVapidPrivateKey, err := decodeVapidKey(vapidPrivateKey)
	if err != nil {
		return "", err
	}

	privKey, err := generateVAPIDHeaderKeys(decodedVapidPrivateKey)
	if err != nil {
		return "", err
	}

	// Sign token with private key
	jwtString, err := token.SignedString(privKey)
	if err != nil {
		return "", err
	}

	// Decode the VAPID public key
	pubKey, err := decodeVapidKey(vapidPublicKey)
	if err != nil {
		return "", err
	}

	return "vapid t=" + jwtString + ", k=" + base64.RawURLEncoding.EncodeToString(pubKey), nil
}

// Need to decode the vapid private key in multiple base64 formats
// Solution from: https://github.com/SherClockHolmes/webpush-go/issues/29
func decodeVapidKey(key string) ([]byte, error) {
	bytes, err := base64.URLEncoding.DecodeString(key)
	if err == nil {
		return bytes, nil
	}

	return base64.RawURLEncoding.DecodeString(key)
}
