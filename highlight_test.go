package logview

import "testing"

const (
	timeRe     = `(?P<SeaGreen>\d{1,2}(?::\d{2})+(?:\.\d{1,5})+)`
	groupRe    = `(?:\[(?P<PowderBlue>.*)\])`
	levelRe    = `(?:\s+(?P<LemonChiffon>info|warn|error|trace|debug)\s+)`
	keyValueRe = `(?P<SkyBlue>[\p{L}]+)(?:=|:)(?P<SteelBlue>(?:".*"|'.*'|[\p{L}\d.]+))`
	numberRe   = `(?:\b(?P<Violet>\d[\d.]+)\b)`
	stringRe   = `(?P<Turquoise>(?:".*")|(?:'.*'))`

	highlightRe = timeRe + "|" + groupRe + "|" + levelRe + "|" + keyValueRe + "|" + numberRe + "|" + stringRe
)

func TestLogView_colorizeGroups(t *testing.T) {
	lv := NewLogView()
	lv.SetHighlightCurrentEvent(true)
	lv.SetHighlightPattern(`\s+(?P<word1>[\p{L}]*)\s+(?P<word2>.*)\s+(?P<num>\d+) (?P<word3>[\p{L}]*)`)

	msg := " Два wordoслова 11 møøsè"
	event := &logEventLine{
		EventID: "1",
		Runes:   []rune(msg),
	}

	lv.colorize(event)

	getSpan := func(i int) string {
		return string(event.Runes[event.styleSpans[i].start:event.styleSpans[i].end])
	}

	expected := map[int]string{
		0: " ",
		1: "Два",
		2: " ",
		3: "wordoслова",
		4: " ",
		5: "11",
		6: " ",
		7: "møøsè",
	}

	if len(event.styleSpans) != len(expected) {
		t.Errorf("Invalid number of spans, expected 3, got %d", len(event.styleSpans))
	}

	for k, v := range expected {
		if getSpan(k) != v {
			t.Errorf("Invalid span %d, expected '%s', got '%s'", k, v, getSpan(k))
		}
	}
}

func TestLogView_colorizeRealWorld(t *testing.T) {
	line := `16:39:41.458 [sqsTaskExecutor-10] INFO  c.s.d.l.s.s.CopyrightDetectionService - Processing pkg:pypi/dgl@0.6a210306?checksum=sha1:b2a504492bba3e49dd3d6433c4e38a087786a4fa&qualifier=cp37-cp37m-manylinux1_x86_64&sonatype_repository=pypi.python.org&type=whl`
	defer func() {
		if state := recover(); state != nil {
			t.Errorf("Failed with panic: %v", state)
		}
	}()

	lv := NewLogView()
	lv.SetHighlighting(true)
	lv.SetHighlightPattern(highlightRe)

	lv.colorize(&logEventLine{
		Runes: []rune(line),
	})

}
