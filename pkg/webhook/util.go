package webhook

import "fmt"

const (
	unit = 1024
)

// ByteCount converts bytes into corresponding unit
func ByteCount(b uint64) string {
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, index := uint64(unit), 0
	for val := b / unit; val >= unit; val /= unit {
		div *= unit
		index++
	}
	return fmt.Sprintf("%d%c",
		uint64(b)/uint64(div), "KMGTPE"[index])
}
