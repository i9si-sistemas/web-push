package webpush

import (
	"context"
	"net/http"
	"testing"

	"github.com/i9si-sistemas/assert"
	"github.com/i9si-sistemas/nine"
)

var (
	webPushClient = New(&testHTTPClient{})
	subscriber    = "gopher@noreply.com"
)

func TestSendNotificationToURLEncodedSubscription(t *testing.T) {
	privateKey, publicKey, err := webPushClient.GenerateVAPIDKeys()
	assert.NoError(t, err)
	resp, err := webPushClient.SendNotification([]byte("Test"), getURLEncodedTestSubscription(), &Options{
		RecordSize:      3070,
		Subscriber:      subscriber,
		Topic:           "test_topic",
		TTL:             0,
		Urgency:         "low",
		VAPIDPublicKey:  publicKey,
		VAPIDPrivateKey: privateKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, resp.StatusCode, http.StatusCreated)
}

func TestSendNotificationToStandardEncodedSubscription(t *testing.T) {
	privateKey, publicKey, err := webPushClient.GenerateVAPIDKeys()
	assert.NoError(t, err)
	resp, err := webPushClient.SendNotification([]byte("Test"), getStandardEncodedTestSubscription(), &Options{
		Subscriber:      subscriber,
		Topic:           "test_topic",
		TTL:             0,
		Urgency:         "low",
		VAPIDPrivateKey: privateKey,
		VAPIDPublicKey:  publicKey,
	})
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusCreated)
}

type testHTTPClient struct{}

func (*testHTTPClient) Post(url string, options *nine.Options) (*http.Response, error) {
	return &http.Response{StatusCode: 201}, nil
}

func (*testHTTPClient) Get(url string, options *nine.Options) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNotImplemented}, nil
}
func (*testHTTPClient) Put(url string, options *nine.Options) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNotImplemented}, nil
}

func (*testHTTPClient) Patch(url string, options *nine.Options) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNotImplemented}, nil
}

func (*testHTTPClient) Delete(url string, options *nine.Options) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusNotImplemented}, nil
}

func (*testHTTPClient) Context() context.Context {
	return context.Background()
}

func getURLEncodedTestSubscription() *Subscription {
	return &Subscription{
		Endpoint: "https://updates.push.services.mozilla.com/wpush/v2/gAAAAA",
		Keys: Keys{
			P256dh: "BNNL5ZaTfK81qhXOx23-wewhigUeFb632jN6LvRWCFH1ubQr77FE_9qV1FuojuRmHP42zmf34rXgW80OvUVDgTk",
			Auth:   "zqbxT6JKstKSY9JKibZLSQ",
		},
	}
}

func getStandardEncodedTestSubscription() *Subscription {
	return &Subscription{
		Endpoint: "https://updates.push.services.mozilla.com/wpush/v2/gAAAAA",
		Keys: Keys{
			P256dh: "BNNL5ZaTfK81qhXOx23+wewhigUeFb632jN6LvRWCFH1ubQr77FE/9qV1FuojuRmHP42zmf34rXgW80OvUVDgTk=",
			Auth:   "zqbxT6JKstKSY9JKibZLSQ==",
		},
	}
}
