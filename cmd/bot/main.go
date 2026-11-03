package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"elayan.local/desa/config"
	"elayan.local/desa/internal/client"
	"elayan.local/desa/internal/gemini"
	"elayan.local/desa/internal/handler"
	"elayan.local/desa/internal/messenger/whatsmeow"
	"elayan.local/desa/internal/repository"
	"elayan.local/desa/internal/service"
	"elayan.local/desa/internal/storage"
	"github.com/mdp/qrterminal/v3"
	wa "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

func main() {
	logger := log.New(os.Stdout, "elayanbot ", log.LstdFlags|log.LUTC)
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("configuration error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dbCtx, cancelDB := context.WithTimeout(ctx, 15*time.Second)
	defer cancelDB()
	db, err := repository.Open(dbCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed; check DATABASE_URL and database availability")
	}
	defer db.Close()

	container := sqlstore.NewWithDB(db, "postgres", nil)
	if err := container.Upgrade(ctx); err != nil {
		logger.Fatalf("WhatsApp session store upgrade failed: %v", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		logger.Fatalf("WhatsApp device session could not be loaded: %v", err)
	}
	waClient := wa.NewClient(device, nil)
	adapter := whatsmeow.New(waClient)
	var qr <-chan wa.QRChannelItem
	if device.ID == nil {
		qr, err = waClient.GetQRChannel(ctx)
		if err != nil {
			logger.Fatalf("WhatsApp pairing could not start: %v", err)
		}
	}
	if err := waClient.ConnectContext(ctx); err != nil {
		logger.Fatalf("WhatsApp connection failed: %v", err)
	}
	if qr != nil {
		logger.Println("Scan this QR code from WhatsApp > Linked devices:")
		for item := range qr {
			if item.Event == "code" {
				qrterminal.Generate(item.Code, qrterminal.M, os.Stdout)
				continue
			}
			if item.Error != nil {
				logger.Printf("WhatsApp pairing failed: %v", item.Error)
				return
			}
		}
	}

	files, err := storage.NewLocal(cfg.StorageDir)
	if err != nil {
		logger.Fatalf("private storage setup failed: %v", err)
	}
	backend, err := client.NewBackend(cfg.BotAPIURL, cfg.BotAPIToken, files)
	if err != nil {
		logger.Fatalf("bot API setup failed: %v", err)
	}
	assistant := gemini.NewAssistant(gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel), cfg.VillageName, cfg.GeminiPerHour, logger)
	if assistant.Enabled() {
		logger.Printf("Gemini assistant enabled (model %s, %d questions/hour/resident)", cfg.GeminiModel, cfg.GeminiPerHour)
	}
	router := handler.NewRouter(db, adapter, backend, files, cfg.RetentionDays, logger, assistant)
	router.Start(ctx)
	go service.RunNotifications(ctx, db, adapter, logger)
	logger.Println("WhatsApp bot connected and waiting for resident messages")
	<-ctx.Done()
	waClient.Disconnect()
}
