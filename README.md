# Web Push Notification Library 

This library provides a simple and efficient way to implement web push notifications in Go applications. It handles VAPID (Voluntary Application Server Identification) authentication and supports various notification urgency levels.

## Installation

To install the library, use:

```bash
go get github.com/i9si-sistemas/web-push
```

## Usage

```go
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/i9si-sistemas/nine"
	i9 "github.com/i9si-sistemas/nine/pkg/server"
	webpush "github.com/i9si-sistemas/web-push"
	"github.com/username/repository/database"
)

type ApiNotification struct {
	SenderId     string `json:"senderId"`
	Subscription struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			Auth   string `json:"auth"`
			P256dh string `json:"p256dh"`
		}
	} `json:"subscription"`
}

var (
	vapidPublicKey  = ""
	vapidPrivateKey = ""
)

type SubscriptionMessage struct {
	SenderId string `json:"senderId"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

func main() {
	ctx := context.Background()
	webpushClient := webpush.New(nine.New(ctx))
	server := nine.NewServer(5502)

	db := database.New()

	server.ServeFiles("/", "./static")

	server.Get("/public_key", func(req *i9.Request, res *i9.Response) error {
		privateKey, publicKey, _ := webpushClient.GenerateVAPIDKeys()
		vapidPrivateKey = privateKey
		vapidPublicKey = publicKey
		return res.JSON(nine.JSON{
			"public_key": publicKey,
		})
	})

	server.Post("/subscribe", func(req *i9.Request, res *i9.Response) error {
		var subscription ApiNotification
		if err := i9.Body(req, &subscription); err != nil {
			return res.Status(http.StatusBadRequest).JSON(nine.JSON{
				"status":  "error",
				"message": err.Error(),
			})
		}
		log.Println(subscription)
		if err := db.Add(database.Subscription{
			SenderId: subscription.SenderId,
			Subscription: webpush.Subscription{
				Endpoint: subscription.Subscription.Endpoint,
				Keys:     webpush.Keys(subscription.Subscription.Keys),
			},
		}); err != nil {
			return res.SendStatus(http.StatusInternalServerError)
		}
		return res.JSON(nine.JSON{
			"status": "ok",
		})
	})

	server.Post("/send/message", func(req *i9.Request, res *i9.Response) error {
		var body struct {
			SenderId string `json:"senderId"`
		}
		if err := i9.Body(req, &body); err != nil {
			return res.Status(http.StatusBadRequest).JSON(nine.JSON{
				"status":  "error",
				"message": err.Error(),
			})
		}
		msg := SubscriptionMessage{
			SenderId: body.SenderId,
			Title:    "Nova Mensagem",
			Body:     "Hello World",
		}
		b, err := json.Marshal(&msg)
		if err != nil {
			log.Println(err)
			return res.SendStatus(http.StatusInternalServerError)
		}
		webpushSubscription, err := db.FindBySenderId(body.SenderId)
		if err != nil {
			return res.SendStatus(http.StatusNotFound)
		}
		subscription := webpush.Subscription{
			Endpoint: webpushSubscription.Endpoint,
			Keys:     webpush.Keys(webpushSubscription.Keys),
		}
		resp, err := webpushClient.SendNotification(b, &subscription, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			return res.SendStatus(http.StatusInternalServerError)
		}
		b, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return res.SendStatus(http.StatusInternalServerError)
		}
		log.Println(string(b))
		return res.JSON(nine.JSON{
			"status": resp.Status,
		})
	})

	server.Listen()
}
```


## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
3. Commit your changes
5. Create a Pull Request

## License

This library is licensed under [License Name]. See the LICENSE file for details.