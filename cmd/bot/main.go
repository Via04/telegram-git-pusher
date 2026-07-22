package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-git-pusher/config"
	botPkg "telegram-git-pusher/internal/bot"
	"telegram-git-pusher/internal/git"

	tele "gopkg.in/telebot.v3"
)

func main() {
	tokenFlag := flag.String("token", "", "Telegram Bot Token")
	dryRunFlag := flag.Bool("dry-run", false, "Enable dry-run mode (commit locally without pushing to remote)")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *tokenFlag != "" {
		cfg.TelegramToken = *tokenFlag
	}
	if *dryRunFlag {
		cfg.DryRun = true
	}

	if cfg.TelegramToken == "" {
		log.Println("⚠️ TELEGRAM_BOT_TOKEN environment variable or -token flag is missing!")
		log.Println("Set TELEGRAM_BOT_TOKEN=your_token in .env or environment, or pass -token flag.")
		log.Println("Run with -help for options.")
		os.Exit(1)
	}

	// Create temp directory for workspaces
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		log.Fatalf("Failed to create work directory %s: %v", cfg.WorkDir, err)
	}

	// Initialize Services
	gitSvc := git.NewGitService(cfg.DefaultGitName, cfg.DefaultGitEmail, cfg.DryRun)
	sessionMgr := botPkg.NewSessionManager(cfg.WorkDir)

	// Create Telebot Bot
	pref := tele.Settings{
		Token:  cfg.TelegramToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("Failed to initialize Telebot: %v", err)
	}

	botHandler := botPkg.NewBotHandler(b, cfg, gitSvc, sessionMgr)
	botHandler.RegisterHandlers()

	log.Printf("🤖 Telegram Git Pusher Bot started successfully!")
	log.Printf("📂 Working directory: %s", cfg.WorkDir)
	if cfg.DryRun {
		log.Printf("🔒 DRY RUN MODE ENABLED: Bot will commit locally and skip remote push.")
	} else {
		log.Printf("🚀 LIVE PUSH MODE: Bot will push commits to remote repositories.")
	}
	if len(cfg.AllowedUsers) > 0 {
		log.Printf("🛡 Allowed Users: %v", cfg.AllowedUsers)
	} else {
		log.Printf("⚠️ No ALLOWED_USERS specified. Bot is accessible to anyone who has its link.")
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		b.Start()
	}()

	<-stopChan
	log.Println("Stopping bot gracefully...")
	b.Stop()
	log.Println("Bot stopped.")
}
