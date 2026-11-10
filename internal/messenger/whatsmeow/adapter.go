package whatsmeow

import (
	"context"
	"fmt"
	"strings"

	"elayan.local/desa/internal/messenger"
	wa "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Adapter struct{ client *wa.Client }

func New(client *wa.Client) *Adapter { return &Adapter{client: client} }

func (a *Adapter) SendText(ctx context.Context, jid, text string) error {
	chat, err := directJID(jid)
	if err != nil {
		return err
	}
	_, err = a.client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(text)})
	return err
}

func (a *Adapter) Typing(ctx context.Context, jid string) error {
	chat, err := directJID(jid)
	if err != nil {
		return err
	}
	return a.client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
}

func (a *Adapter) SendFile(ctx context.Context, jid, filename, mime string, data []byte) error {
	chat, err := directJID(jid)
	if err != nil {
		return err
	}
	upload, err := a.client.Upload(ctx, data, wa.MediaDocument)
	if err != nil {
		return err
	}
	document := &waE2E.DocumentMessage{
		URL:           &upload.URL,
		DirectPath:    &upload.DirectPath,
		MediaKey:      upload.MediaKey,
		FileEncSHA256: upload.FileEncSHA256,
		FileSHA256:    upload.FileSHA256,
		FileLength:    &upload.FileLength,
		Mimetype:      proto.String(mime),
		FileName:      proto.String(filename),
	}
	_, err = a.client.SendMessage(ctx, chat, &waE2E.Message{DocumentMessage: document})
	return err
}

func (a *Adapter) DownloadMedia(ctx context.Context, media messenger.Media) ([]byte, error) {
	message, ok := media.Payload.(wa.DownloadableMessage)
	if !ok || media.Size == 0 || media.Size > 10<<20 {
		return nil, fmt.Errorf("unsupported or oversized WhatsApp media")
	}
	data, err := a.client.Download(ctx, message)
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) > 10<<20 {
		return nil, fmt.Errorf("WhatsApp media exceeds limit")
	}
	return data, nil
}

func (a *Adapter) OnMessage(handler func(messenger.Message)) {
	a.client.AddEventHandler(func(event any) {
		incoming, ok := event.(*events.Message)
		if !ok || incoming.Info.IsFromMe || incoming.Info.IsGroup {
			return
		}
		message := incoming.UnwrapRaw()
		if message == nil || message.Message == nil {
			return
		}
		phoneJID := ""
		if incoming.Info.Chat.Server == types.DefaultUserServer {
			phoneJID = incoming.Info.Chat.String()
		} else if incoming.Info.SenderAlt.Server == types.DefaultUserServer {
			phoneJID = incoming.Info.SenderAlt.String()
		}
		if phoneJID == "" {
			return
		}
		text := message.Message.GetConversation()
		if extended := message.Message.GetExtendedTextMessage(); extended != nil {
			text = extended.GetText()
		}
		result := messenger.Message{JID: incoming.Info.Chat.String(), PhoneJID: phoneJID, Text: strings.TrimSpace(text)}
		if image := message.Message.GetImageMessage(); image != nil {
			result.Text = strings.TrimSpace(image.GetCaption())
			result.Media = &messenger.Media{MIME: image.GetMimetype(), Size: image.GetFileLength(), Payload: image}
		}
		handler(result)
	})
}


func directJID(raw string) (types.JID, error) {
	jid, err := types.ParseJID(raw)
	if err != nil || jid.Server != types.DefaultUserServer && jid.Server != types.HiddenUserServer {
		return types.JID{}, fmt.Errorf("invalid direct WhatsApp JID")
	}
	return jid, nil
}