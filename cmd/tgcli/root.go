package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/GodsBoy/tgcli/internal/client"
	"github.com/GodsBoy/tgcli/internal/config"
	"github.com/GodsBoy/tgcli/internal/lock"
	"github.com/GodsBoy/tgcli/internal/store"
)

type rootFlags struct {
	storeDir string
	asJSON   bool
	timeout  time.Duration
}

func execute(args []string) error {
	var flags rootFlags

	rootCmd := &cobra.Command{
		Use:     "tgcli",
		Short:   "Telegram CLI - sync, search, and send messages via MTProto",
		Version: version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&flags.storeDir, "store", "", "storage directory (default: ~/.tgcli)")
	rootCmd.PersistentFlags().BoolVar(&flags.asJSON, "json", false, "output as JSON")
	rootCmd.PersistentFlags().DurationVar(&flags.timeout, "timeout", 5*time.Minute, "operation timeout")

	rootCmd.AddCommand(newAuthCmd(&flags))
	rootCmd.AddCommand(newSyncCmd(&flags))
	rootCmd.AddCommand(newMessagesCmd(&flags))
	rootCmd.AddCommand(newSendCmd(&flags))
	rootCmd.AddCommand(newChatsCmd(&flags))
	rootCmd.AddCommand(newContactsCmd(&flags))
	rootCmd.AddCommand(newGroupsCmd(&flags))
	rootCmd.AddCommand(newDoctorCmd(&flags))

	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func resolveStoreDir(flags *rootFlags) string {
	if flags.storeDir != "" {
		return flags.storeDir
	}
	if dir := os.Getenv("TGCLI_STORE_DIR"); dir != "" {
		return dir
	}
	return config.DefaultStoreDir()
}

type appContext struct {
	db       *store.DB
	client   *client.Client
	lk       *lock.Lock
	storeDir string
	cfg      config.Config
}

func newAppContext(ctx context.Context, flags *rootFlags, needLock bool) (*appContext, error) {
	storeDir := resolveStoreDir(flags)

	if err := os.MkdirAll(storeDir, 0700); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	ac := &appContext{storeDir: storeDir}

	if needLock {
		lk, err := lock.Acquire(storeDir)
		if err != nil {
			return nil, err
		}
		ac.lk = lk
	}

	// Open database.
	dbPath := filepath.Join(storeDir, "tgcli.db")
	db, err := store.Open(dbPath)
	if err != nil {
		ac.close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	ac.db = db

	// Load config.
	cfgPath := filepath.Join(storeDir, "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		ac.close()
		return nil, fmt.Errorf("load config: %w", err)
	}
	ac.cfg = cfg

	// Override config from env vars.
	if v := os.Getenv("TGCLI_APP_ID"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			ac.cfg.AppID = id
		}
	}
	if v := os.Getenv("TGCLI_APP_HASH"); v != "" {
		ac.cfg.AppHash = v
	}
	if v := os.Getenv("TGCLI_PHONE"); v != "" {
		ac.cfg.Phone = v
	}

	return ac, nil
}

func (ac *appContext) initClient() error {
	c, err := client.New(client.Options{
		AppID:    ac.cfg.AppID,
		AppHash:  ac.cfg.AppHash,
		StoreDir: ac.storeDir,
	})
	if err != nil {
		return err
	}
	ac.client = c
	return nil
}

func (ac *appContext) close() {
	if ac.db != nil {
		ac.db.Close()
	}
	if ac.lk != nil {
		ac.lk.Release()
	}
}
