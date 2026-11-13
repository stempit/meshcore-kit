package fit

import (
	"strings"
	"unicode/utf8"
)

// cutOnBoundary returns the largest index ≤ limit that lands on a UTF-8 rune
// boundary, preferring a sentence end (". " / "! " / "? " — terminator
// followed by a space) in the back half of the range so the text after the
// cut begins with the next sentence, and falling back to a plain space so
// cuts land on word boundaries. A decimal dot ("18.7km") has no following
// space and never matches the sentence rule.
func cutOnBoundary(s string, limit int) int {
	if limit >= len(s) {
		return len(s)
	}
	if limit < 0 {
		return 0
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	best := -1
	for _, term := range [...]string{". ", "! ", "? "} {
		if idx := strings.LastIndex(s[:cut], term); idx > best {
			best = idx
		}
	}
	if best >= 0 && best+1 > limit/2 {
		return best + 1 // keep the terminator; the space starts the next cut
	}
	if idx := strings.LastIndex(s[:cut], " "); idx > limit/2 {
		cut = idx
	}
	return cut
}

// truncate cuts s to at most max bytes on a rune boundary, appending the
// ellipsis (which also counts toward max).
func truncate(s string, max int, ellipsis string) string {
	if len(s) <= max {
		return s
	}
	limit := max - len(ellipsis)
	if limit < 0 {
		limit = 0
		ellipsis = ""
	}
	return strings.TrimSpace(s[:cutOnBoundary(s, limit)]) + ellipsis
}

// hardCap is the final, ellipsis-free safety cut applied to every returned
// message. It is a no-op unless a message is still over max (e.g. a prefix
// longer than the budget).
func hardCap(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:cutOnBoundary(s, max)]
}
