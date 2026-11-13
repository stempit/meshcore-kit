package fit

import (
	"fmt"
	"strings"
)

const (
	// maxParts caps a single split message at 9 wire parts.
	maxParts = 9
	// markerReserve is the byte cost of the largest series marker, " [9/9]".
	// Since N ≤ 9 the marker is always single-digit on both sides.
	markerReserve = 6
)

// SplitParts is the multi-part fallback: it squeezes text per o and splits it
// into up to 9 rune-safe "[i/N]" parts. Exported so injected Compressors can
// fall back to it.
func SplitParts(o Options, text string, head int) []string {
	return split(o.Squeeze(text), o.Max(), head, o.Ellipsis)
}

// split breaks text into up to maxParts rune-safe chunks, each suffixed with a
// "[i/N]" series marker, every part fitting within max bytes. head reserves
// bytes on the first chunk only (for a send-layer prefix). The text's own
// structure is respected: whole "\n"-separated lines are packed per chunk and
// a chunk break never falls mid-line — the next chunk begins with the
// following line. A single line longer than the chunk budget (or the
// maxParts overflow) is cut sentence-first, then on a space, via
// cutOnBoundary.
func split(text string, max, head int, ellipsis string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var chunks []string
	cur := "" // current chunk: whole lines joined by "\n"
	// budget returns the byte budget for the chunk about to be filled: the
	// first chunk also carries the send-layer prefix.
	budget := func() int {
		b := max - markerReserve
		if len(chunks) == 0 {
			b -= head
		}
		if b < 1 {
			b = 1
		}
		return b
	}
	add := func(s string) {
		chunks = append(chunks, s)
		cur = ""
	}

	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		// Last allowed part with text remaining: truncate everything left
		// (the packed lines plus the rest) to fit.
		if len(chunks) == maxParts-1 {
			rest := strings.Join(lines[i:], "\n")
			if cur != "" {
				rest = cur + "\n" + rest
			}
			add(truncate(strings.TrimSpace(rest), budget(), ellipsis))
			return mark(chunks)
		}
		b := budget()
		if cur == "" && len(line) > b {
			// One line alone overflows the empty chunk: cut it (sentence /
			// space aware) and let the remainder flow into the next chunk.
			cut := cutOnBoundary(line, b)
			if cut <= 0 {
				cut = b
			}
			add(strings.TrimSpace(line[:cut]))
			lines[i] = strings.TrimSpace(line[cut:])
			continue
		}
		candidate := line
		if cur != "" {
			candidate = cur + "\n" + line
		}
		if len(candidate) <= b {
			cur = candidate
			i++
			continue
		}
		add(cur) // the chunk is full; the line opens the next one
	}
	if cur != "" {
		add(cur)
	}

	return mark(chunks)
}

// mark appends the "[i/N]" series marker to every chunk; a single chunk is
// returned unmarked.
func mark(chunks []string) []string {
	if len(chunks) <= 1 {
		return chunks
	}
	total := len(chunks)
	for i := range chunks {
		chunks[i] = fmt.Sprintf("%s [%d/%d]", chunks[i], i+1, total)
	}
	return chunks
}
