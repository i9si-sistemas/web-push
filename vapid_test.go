package webpush

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/i9si-sistemas/assert"
)

func TestVAPID(t *testing.T) {
	s := getStandardEncodedTestSubscription()
	sub := "test@test.com"

	vapidPrivateKey, vapidPublicKey, err := webPushClient.GenerateVAPIDKeys()
	assert.NoError(t, err)
	vapidAuthHeader, err := getVAPIDAuthorizationHeader(
		s.Endpoint,
		sub,
		vapidPublicKey,
		vapidPrivateKey,
		time.Now().Add(time.Hour*12),
	)
	assert.NoError(t, err)

	tokenString := getTokenFromAuthorizationHeader(t, vapidAuthHeader)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodECDSA)
		assert.True(t, ok, "Wrong validation method need ECDSA!")

		b64 := base64.RawURLEncoding
		decodedVapidPrivateKey, err := b64.DecodeString(vapidPrivateKey)
		assert.NoError(t, err)

		privKey, err := generateVAPIDHeaderKeys(decodedVapidPrivateKey)
		assert.NoError(t, err)
		return privKey.Public(), nil
	})
	assert.NoError(t, err)

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		expectedSub := fmt.Sprintf("mailto:%s", sub)
		assert.Equal(t, claims["sub"], expectedSub)
		assert.NotEmpty(t, claims["aud"])
	}

}

func TestVAPIDKeys(t *testing.T) {
	privateKey, publicKey, err := webPushClient.GenerateVAPIDKeys()
	assert.NoError(t, err)
	assert.Equal(t, len(privateKey), 43)
	assert.Equal(t, len(publicKey), 87)
}

// Helper function for extracting the token from the Authorization header
func getTokenFromAuthorizationHeader(t assert.T, tokenHeader string) string {
	hsplit := strings.Split(tokenHeader, " ")
	ok := len(hsplit) < 3
	assert.False(t, ok)

	tsplit := strings.Split(hsplit[1], "=")
	ok = len(tsplit) < 2
	assert.False(t, ok)

	return tsplit[1][:len(tsplit[1])-1]
}
