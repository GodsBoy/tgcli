package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/GodsBoy/tgcli/internal/format"
)

func newDoctorCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			storeDir := resolveStoreDir(flags)

			checks := []struct {
				name   string
				status string
				detail string
			}{
				{
					name:   "version",
					status: "ok",
					detail: version,
				},
				{
					name:   "go_version",
					status: "ok",
					detail: runtime.Version(),
				},
				{
					name:   "store_dir",
					status: checkDir(storeDir),
					detail: storeDir,
				},
				{
					name:   "config",
					status: checkFile(filepath.Join(storeDir, "config.json")),
					detail: filepath.Join(storeDir, "config.json"),
				},
				{
					name:   "session",
					status: checkFile(filepath.Join(storeDir, "session.json")),
					detail: filepath.Join(storeDir, "session.json"),
				},
				{
					name:   "database",
					status: checkFile(filepath.Join(storeDir, "tgcli.db")),
					detail: filepath.Join(storeDir, "tgcli.db"),
				},
				{
					name:   "env_app_id",
					status: checkEnv("TGCLI_APP_ID"),
					detail: "TGCLI_APP_ID",
				},
				{
					name:   "env_app_hash",
					status: checkEnv("TGCLI_APP_HASH"),
					detail: "TGCLI_APP_HASH",
				},
			}

			// Check FTS5 support.
			dbPath := filepath.Join(storeDir, "tgcli.db")
			ac, err := newAppContext(cmd.Context(), flags, false)
			ftsStatus := "unknown"
			if err == nil {
				if ac.db.FTSEnabled() {
					ftsStatus = "enabled"
				} else {
					ftsStatus = "disabled"
				}
				ac.close()
			}
			checks = append(checks, struct {
				name   string
				status string
				detail string
			}{
				name:   "fts5",
				status: ftsStatus,
				detail: dbPath,
			})

			if flags.asJSON {
				result := make([]map[string]string, len(checks))
				for i, c := range checks {
					result[i] = map[string]string{
						"name":   c.name,
						"status": c.status,
						"detail": c.detail,
					}
				}
				return format.WriteJSON(os.Stdout, result)
			}

			for _, c := range checks {
				icon := "✓"
				if c.status != "ok" && c.status != "enabled" {
					icon = "✗"
				}
				fmt.Printf("  %s %-15s %s (%s)\n", icon, c.name, c.status, c.detail)
			}
			return nil
		},
	}
}

func checkDir(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	if !info.IsDir() {
		return "not_a_directory"
	}
	return "ok"
}

func checkFile(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	if info.IsDir() {
		return "is_a_directory"
	}
	return "ok"
}

func checkEnv(name string) string {
	if os.Getenv(name) != "" {
		return "ok"
	}
	return "not_set"
}
