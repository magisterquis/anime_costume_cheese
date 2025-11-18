//go:build linux && !no_sneaky_remap

package main

/*
 * sneaky_remap.go
 * Add in sneaky_remap on Linux unless we're told to not
 * By J. Stuart McMurray
 * Created 20251116
 * Last Modified 20251118
 */

import _ "github.com/magisterquis/sneaky_remap"
