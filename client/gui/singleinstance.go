package main

import (
	"errors"

	"golang.org/x/sys/windows"
)

// singleInstanceMutexName is process-wide and unique enough not to collide
// with anything else on the system.
const singleInstanceMutexName = `Global\RookeryGUISingleInstance`

// acquireSingleInstanceLock reports whether this is the only running
// instance of the app. The returned handle must be kept alive (never
// garbage collected/closed) for the lock to hold until the process exits —
// callers should just let it leak for the process lifetime.
func acquireSingleInstanceLock() bool {
	name, err := windows.UTF16PtrFromString(singleInstanceMutexName)
	if err != nil {
		return true // fail open rather than block startup over this
	}

	_, err = windows.CreateMutex(nil, false, name)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return true // fail open
	}
	return !errors.Is(err, windows.ERROR_ALREADY_EXISTS)
}
