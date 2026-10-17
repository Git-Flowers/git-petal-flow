// Package events writes the per-branch JSONL journal.
package events

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewOpID returns an operation ID. The 16-digit hex timestamp prefix makes IDs
// sort by creation time; the random suffix keeps them unique within a
// nanosecond.
func NewOpID(now time.Time) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("op_%016x%s", now.UnixNano(), hex.EncodeToString(b[:])), nil
}
