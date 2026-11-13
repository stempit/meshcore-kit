package fit

import (
	"strings"
	"testing"
)

// TestSplitPacksWholeLines pins the newline rule: chunk breaks fall between
// "\n"-separated lines, never mid-line — the next part begins with the
// following line even when a shorter word-level cut would fit more bytes.
func TestSplitPacksWholeLines(t *testing.T) {
	text := strings.Join([]string{
		"alpha line",      // 10
		"beta line here",  // 14
		"gamma line",      // 10
		"delta line ends", // 15
	}, "\n")
	out := Fit(Options{MaxBytes: 30, Variant: Split}, "", []string{text})
	if len(out) < 2 {
		t.Fatalf("expected multiple parts, got %v", out)
	}
	for i, part := range out {
		body := markerRe.ReplaceAllString(part, "")
		for _, line := range strings.Split(body, "\n") {
			switch line {
			case "alpha line", "beta line here", "gamma line", "delta line ends":
			default:
				t.Errorf("part %d contains a cut line %q", i, line)
			}
		}
	}
}

// TestSplitTraceLinesReplay replays the report that motivated the change: a
// multi-line path report where the old splitter cut
// "[KRV...] (5.2c) 184м+" across two wire messages. Whole lines must survive.
func TestSplitTraceLinesReplay(t *testing.T) {
	text := strings.Join([]string{
		"пyтeй: 6 зa 15c",
		"[direct] (0.0c)",
		"[LETO] (1.0c) 2.2км+",
		"[Ipodrom] (1.6c) 220м+",
		"[KRV_Ippodrom_North] (5.2c) 184м+",
		"[KRV_Ippodrom_South] (6.4c) 181м+",
		"[LETO][CБ][EDDS-Ganino][REPA] (9.2c) 26.0км+",
	}, "\n")
	out := Fit(Options{MaxBytes: max, Variant: Split}, "@[faketest1] ", []string{text})
	checkAllFit(t, out, "@[faketest1] ")
	for _, part := range out {
		body := markerRe.ReplaceAllString(strings.TrimPrefix(part, "@[faketest1] "), "")
		for _, line := range strings.Split(body, "\n") {
			// Every line is a complete path entry: it either starts with a
			// label or IS the header — never a bare continuation fragment.
			if !strings.HasPrefix(line, "[") && !strings.HasPrefix(line, "пyтeй:") {
				t.Errorf("part %q contains a mid-line fragment %q", part, line)
			}
		}
	}
}

// TestSplitBreaksAtSentenceEnd pins the sentence rule: when a single long
// line must be cut, the cut lands right after a sentence terminator and the
// next part begins with the following sentence.
func TestSplitBreaksAtSentenceEnd(t *testing.T) {
	text := "First sentence is here. Second sentence follows it. Third sentence closes."
	out := Fit(Options{MaxBytes: 40, Variant: Split}, "", []string{text})
	if len(out) < 2 {
		t.Fatalf("expected multiple parts, got %v", out)
	}
	if body := markerRe.ReplaceAllString(out[0], ""); !strings.HasSuffix(body, "here.") {
		t.Errorf("part 1 = %q, want it to end at the sentence boundary", out[0])
	}
	if body := markerRe.ReplaceAllString(out[1], ""); !strings.HasPrefix(body, "Second") {
		t.Errorf("part 2 = %q, want it to open with the next sentence", out[1])
	}
}

// TestSplitSentenceRuleSkipsDecimalDots pins the guard: a decimal dot inside
// a number has no following space, so it must never be mistaken for a
// sentence end.
func TestSplitSentenceRuleSkipsDecimalDots(t *testing.T) {
	text := "Distance reads 18.7km total now and the rest of the words continue past it"
	out := Fit(Options{MaxBytes: 40, Variant: Split}, "", []string{text})
	for _, part := range out {
		if strings.Contains(part, "18.") && !strings.Contains(part, "18.7km") {
			t.Errorf("part %q cuts inside the decimal number", part)
		}
	}
}

// TestSplitEmptyLinesDropped pins that blank separator lines do not produce
// empty wire messages (a text that fits whole is returned untouched — the
// dropping only applies where chunks actually form).
func TestSplitEmptyLinesDropped(t *testing.T) {
	out := Fit(Options{MaxBytes: 30, Variant: Split}, "", []string{"one line here\ntwo line here\n\n\nthree line here"})
	joined := strings.Join(out, "|")
	if strings.Contains(joined, "\n\n") {
		t.Errorf("empty lines leaked into output: %q", joined)
	}
	if !strings.Contains(joined, "three line here") {
		t.Errorf("last line lost: %v", out)
	}
}

// TestSplitKeepsPackedLinesWhenLongLineFollows pins the guard on the
// long-line cut: it fires only on an empty chunk. When an over-long line
// arrives after packed lines, the packed chunk is flushed whole first —
// a cut branch that ignores the guard would reset cur and silently lose
// the packed lines.
func TestSplitKeepsPackedLinesWhenLongLineFollows(t *testing.T) {
	long := "This first sentence is long. Second sentence continues here. Third."
	out := Fit(Options{MaxBytes: 40, Variant: Split}, "", []string{"short line\n" + long})
	if body := markerRe.ReplaceAllString(out[0], ""); body != "short line" {
		t.Errorf("part 1 = %q, want the packed line flushed whole before the long line", out[0])
	}
}

// TestSplitPartsLineAware pins the exported fallback: SplitParts follows the
// same line-packing rules as the split variant.
func TestSplitPartsLineAware(t *testing.T) {
	o := Options{MaxBytes: 30, Variant: Split}
	out := SplitParts(o, "short line\nanother line\nthird line here", 0)
	if len(out) < 2 {
		t.Fatalf("expected multiple parts, got %v", out)
	}
	for i, part := range out {
		body := markerRe.ReplaceAllString(part, "")
		if !strings.HasPrefix(body, "short") && !strings.HasPrefix(body, "another") && !strings.HasPrefix(body, "third") {
			t.Errorf("part %d starts mid-line: %q", i, body)
		}
	}
}
