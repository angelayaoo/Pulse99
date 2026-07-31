package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orbital-sentinel/internal/alerter"
	"orbital-sentinel/internal/api"
	"orbital-sentinel/internal/checker"
	"orbital-sentinel/internal/config"
	"orbital-sentinel/internal/logger"
	"orbital-sentinel/internal/notify"
	"orbital-sentinel/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger.Banner()
	logger.ConfigLoaded(len(cfg.Targets), cfg.IntervalSeconds, cfg.FailureThreshold)

	var notifiers []notify.Notifier
	if cfg.Webhooks.Discord != "" {
		notifiers = append(notifiers, notify.NewDiscord(cfg.Webhooks.Discord))
	}
	if cfg.Webhooks.Telegram.BotToken != "" && cfg.Webhooks.Telegram.ChatID != "" {
		notifiers = append(notifiers, notify.NewTelegram(cfg.Webhooks.Telegram.BotToken, cfg.Webhooks.Telegram.ChatID))
	}
	if cfg.Alerts.Email.Enabled {
		notifiers = append(notifiers, notify.NewEmail(
			cfg.Alerts.Email.SMTPHost,
			cfg.Alerts.Email.SMTPPort,
			cfg.Alerts.Email.SMTPUser,
			cfg.Alerts.Email.SMTPPass,
			cfg.Alerts.Email.FromAddress,
			cfg.Alerts.Email.ToAddresses,
		))
	}
	if cfg.Alerts.Twilio.Enabled {
		notifiers = append(notifiers, notify.NewTwilio(
			cfg.Alerts.Twilio.AccountSID,
			cfg.Alerts.Twilio.AuthToken,
			cfg.Alerts.Twilio.FromPhone,
			cfg.Alerts.Twilio.ToPhones,
		))
	}

	a := alerter.New(notifiers, 300, 3, 2000)

	db, err := store.Open(cfg.Storage.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	engine := checker.NewSweepEngine(cfg.Targets, a, db, cfg.FailureThreshold)

	var apiServer *api.Server
	if cfg.Dashboard.Enabled {
		hub := api.NewHub()
		go hub.Run()
		apiServer = api.NewServer(db, engine, hub, cfg.Dashboard.Port)
		go func() {
			if err := apiServer.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Dashboard error: %v\n", err)
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	iteration := 0

	iteration++
	engine.RunSweep(iteration)
	if apiServer != nil {
		apiServer.BroadcastStatus()
	}

	for {
		select {
		case <-ticker.C:
			iteration++
			engine.RunSweep(iteration)
			if apiServer != nil {
				apiServer.BroadcastStatus()
			}
		case sig := <-sigCh:
			logger.Shutdown()
			_ = sig
			return
		}
	}
}
