package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/i9si-sistemas/nine"
	"golang.org/x/crypto/hkdf"
)

const MaxRecordSize uint32 = 4096

var ErrMaxPadExceeded = errors.New("payload has exceeded the maximum length")

// salt generates a salt of 16 bytes
func salt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return salt, err
	}

	return salt, nil
}

// Options are config and extra params needed to send a notification
type Options struct {
	RecordSize      uint32    // Limit the record size
	Subscriber      string    // Sub in VAPID JWT token
	Topic           string    // Set the Topic header to collapse a pending messages (Optional)
	TTL             int       // Set the TTL on the endpoint POST request
	Urgency         Urgency   // Set the Urgency header to change a message priority (Optional)
	VAPIDPublicKey  string    // VAPID public key, passed in VAPID Authorization header
	VAPIDPrivateKey string    // VAPID private key, used to sign VAPID JWT token
	VapidExpiration time.Time // optional expiration for VAPID JWT token (defaults to now + 12 hours)
}

// Keys are the base64 encoded values from PushSubscription.getKey()
type Keys struct {
	Auth   string `json:"auth"`
	P256dh string `json:"p256dh"`
}

// Subscription represents a PushSubscription object from the Push API
type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     Keys   `json:"keys"`
}

// SendNotification calls SendNotificationWithContext with default context for backwards-compatibility
func (c *Client) SendNotification(message []byte, s *Subscription, options *Options) (*http.Response, error) {
	return c.SendNotificationWithContext(context.Background(), message, s, options)
}

func deriveSharedSecret(subscriberPublicKey []byte) ([]byte, []byte, error) {
	curve := ecdh.P256()

	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	localPublicKey := privateKey.PublicKey().Bytes()

	subKey, err := curve.NewPublicKey(subscriberPublicKey)
	if err != nil {
		return nil, nil, errors.New("invalid subscriber public key")
	}

	sharedSecret, err := privateKey.ECDH(subKey)
	if err != nil {
		return nil, nil, err
	}

	return sharedSecret, localPublicKey, nil
}

// SendNotificationWithContext sends a push notification to a subscription's endpoint
// Message Encryption for Web Push, and VAPID protocols.
// FOR MORE INFORMATION SEE RFC8291: https://datatracker.ietf.org/doc/rfc8291
func (cl *Client) SendNotificationWithContext(
	ctx context.Context,
	message []byte,
	s *Subscription,
	options *Options,
) (*http.Response, error) {
	authSecret, err := decodeSubscriptionKey(s.Keys.Auth)
	if err != nil {
		return nil, err
	}

	dh, err := decodeSubscriptionKey(s.Keys.P256dh)
	if err != nil {
		return nil, err
	}

	salt, err := salt()
	if err != nil {
		return nil, err
	}

	sharedECDHSecret, localPublicKey, err := deriveSharedSecret(dh)
	if err != nil {
		return nil, err
	}

	hash := sha256.New

	prkInfoBuf := bytes.NewBuffer([]byte("WebPush: info\x00"))
	prkInfoBuf.Write(dh)
	prkInfoBuf.Write(localPublicKey)

	prkHKDF := hkdf.New(hash, sharedECDHSecret, authSecret, prkInfoBuf.Bytes())
	ikm, err := getHKDFKey(prkHKDF, 32)
	if err != nil {
		return nil, err
	}

	contentEncryptionKeyInfo := []byte("Content-Encoding: aes128gcm\x00")
	contentHKDF := hkdf.New(hash, ikm, salt, contentEncryptionKeyInfo)
	contentEncryptionKey, err := getHKDFKey(contentHKDF, 16)
	if err != nil {
		return nil, err
	}

	nonceInfo := []byte("Content-Encoding: nonce\x00")
	nonceHKDF := hkdf.New(hash, ikm, salt, nonceInfo)
	nonce, err := getHKDFKey(nonceHKDF, 12)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(contentEncryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	recordSize := options.RecordSize
	if recordSize == 0 {
		recordSize = MaxRecordSize
	}

	recordLength := int(recordSize) - 16

	recordBuf := bytes.NewBuffer(salt)

	rs := make([]byte, 4)
	binary.BigEndian.PutUint32(rs, recordSize)

	recordBuf.Write(rs)
	recordBuf.Write([]byte{byte(len(localPublicKey))})
	recordBuf.Write(localPublicKey)
	dataBuf := bytes.NewBuffer(message)
	paddingEndingDelimeter(
		dataBuf,
		recordBuf,
		recordLength,
		gcm,
		nonce,
	)

	// POST request
	headers, err := notificationHeaders(s.Endpoint, recordBuf, options)
	if err != nil {
		return nil, err
	}
	httpClient := cl.httpClient
	if httpClient == nil {
		httpClient = nine.New(ctx)
	}
	return httpClient.Post(s.Endpoint, &nine.Options{
		Body:    recordBuf,
		Headers: headers,
	})
}

// paddingEndingDelimeter pad the content to max record size
func paddingEndingDelimeter(
	dataBuf *bytes.Buffer,
	recordBuf *bytes.Buffer,
	recordLength int,
	gcm cipher.AEAD,
	nonce []byte,
) error {
	dataBuf.Write([]byte("\x02"))
	if err := pad(dataBuf, recordLength-recordBuf.Len()); err != nil {
		return err
	}
	ciphertext := gcm.Seal([]byte{}, nonce, dataBuf.Bytes(), nil)
	recordBuf.Write(ciphertext)
	return nil
}

// decodeSubscriptionKey decodes a base64 subscription key.
// if necessary, add "=" padding to the key for URL decode
func decodeSubscriptionKey(key string) ([]byte, error) {
	// "=" padding
	buf := bytes.NewBufferString(key)
	if rem := len(key) % 4; rem != 0 {
		buf.WriteString(strings.Repeat("=", 4-rem))
	}

	bytes, err := base64.StdEncoding.DecodeString(buf.String())
	if err == nil {
		return bytes, nil
	}

	return base64.URLEncoding.DecodeString(buf.String())
}

// Returns a key of length "length" given an hkdf function
func getHKDFKey(hkdf io.Reader, length int) ([]byte, error) {
	key := make([]byte, length)
	n, err := io.ReadFull(hkdf, key)
	if n != len(key) || err != nil {
		return key, err
	}

	return key, nil
}

func pad(payload *bytes.Buffer, maxPadLen int) error {
	payloadLen := payload.Len()
	if payloadLen > maxPadLen {
		return ErrMaxPadExceeded
	}

	padLen := maxPadLen - payloadLen

	padding := make([]byte, padLen)
	payload.Write(padding)

	return nil
}

func notificationHeaders(
	endpoint string,
	recordBuf *bytes.Buffer,
	options *Options,
) ([]nine.Header, error) {
	headers := []nine.Header{
		{Data: nine.Data{Key: "Content-Encoding", Value: "aes128gcm"}},
		{Data: nine.Data{Key: "Content-Length", Value: strconv.Itoa(recordBuf.Len())}},
		{Data: nine.Data{Key: "Content-Type", Value: "application/octet-stream"}},
		{Data: nine.Data{Key: "TTL", Value: strconv.Itoa(options.TTL)}},
	}
	expiration := options.VapidExpiration
	if expiration.IsZero() {
		expiration = time.Now().Add(time.Hour * 12)
	}

	vapidAuthHeader, err := getVAPIDAuthorizationHeader(
		endpoint,
		options.Subscriber,
		options.VAPIDPublicKey,
		options.VAPIDPrivateKey,
		expiration,
	)
	if err != nil {
		return nil, err
	}

	headers = addHeader(headers, "Authorization", vapidAuthHeader)

	if len(options.Topic) > 0 {
		headers = addHeader(headers, "Topic", options.Topic)
	}

	if isValidUrgency(options.Urgency) {
		headers = addHeader(headers, "Urgency", string(options.Urgency))
	}
	return headers, nil
}

func addHeader(headers []nine.Header, key string, value any) []nine.Header {
	return append(headers, nine.Header{
		Data: nine.Data{Key: key, Value: value},
	})
}
