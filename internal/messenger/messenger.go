package messenger

import "context"

type Media struct {
	MIME    string
	Size    uint64
	Payload any
}

type Message struct {
	JID      string
	PhoneJID string
	Text     string
	Media    *Media
}

type Messenger interface {
	SendText(context.Context, string, string) error
	SendFile(context.Context, string, string, string, []byte) error
	DownloadMedia(context.Context, Media) ([]byte, error)
	OnMessage(func(Message))
}
