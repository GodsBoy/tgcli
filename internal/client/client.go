package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// Options configures the Telegram client.
type Options struct {
	AppID    int
	AppHash  string
	StoreDir string
}

// Client wraps gotd/td for tgcli.
type Client struct {
	opts   Options
	client *telegram.Client
	waiter *floodwait.Waiter
	api    *tg.Client
}

// New creates a new Telegram client.
func New(opts Options) (*Client, error) {
	if opts.AppID == 0 || opts.AppHash == "" {
		return nil, fmt.Errorf("app_id and app_hash are required (set TGCLI_APP_ID and TGCLI_APP_HASH)")
	}

	sessionPath := filepath.Join(opts.StoreDir, "session.json")
	sessionStorage := &session.FileStorage{Path: sessionPath}

	waiter := floodwait.NewWaiter()

	client := telegram.NewClient(opts.AppID, opts.AppHash, telegram.Options{
		SessionStorage: sessionStorage,
		Middlewares:    []telegram.Middleware{waiter},
	})

	return &Client{
		opts:   opts,
		client: client,
		waiter: waiter,
	}, nil
}

// Run executes fn within an authenticated client session.
// The waiter middleware is started first, then the client connects and runs fn.
func (c *Client) Run(ctx context.Context, fn func(ctx context.Context, client *tg.Client) error) error {
	return c.waiter.Run(ctx, func(ctx context.Context) error {
		return c.client.Run(ctx, func(ctx context.Context) error {
			c.api = c.client.API()
			return fn(ctx, c.api)
		})
	})
}

// API returns the underlying tg.Client. Only valid inside Run().
func (c *Client) API() *tg.Client {
	return c.api
}

// IsAuthed checks if we have a valid session file.
func (c *Client) IsAuthed() bool {
	sessionPath := filepath.Join(c.opts.StoreDir, "session.json")
	info, err := os.Stat(sessionPath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// SessionPath returns the path to the session file.
func (c *Client) SessionPath() string {
	return filepath.Join(c.opts.StoreDir, "session.json")
}
