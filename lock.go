package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const lockTTL = 3 * time.Hour

func lockPath(audioPath string) string {
	return audioPath + ".lock"
}

func acquireLock(audioPath string) error {
	lp := lockPath(audioPath)

	if data, err := os.ReadFile(lp); err == nil {
		ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err == nil && time.Since(time.Unix(ts, 0)) < lockTTL {
			since := time.Since(time.Unix(ts, 0)).Round(time.Second)
			return fmt.Errorf("%s is already being processed (started %s ago)", audioPath, since)
		}
		// stale lock — remove it
		os.Remove(lp)
	}

	return os.WriteFile(lp, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}

func releaseLock(audioPath string) {
	os.Remove(lockPath(audioPath))
}
