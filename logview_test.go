package logview

import (
	"github.com/gdamore/tcell/v2"
	"strconv"
	"testing"
	"time"
)

func TestLogView_EventCount(t *testing.T) {
	lv := NewLogView()
	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	if lv.EventCount() != 100 {
		t.Errorf("Event count is not correct")
	}
}

func TestLogView_ScrollToEventID(t *testing.T) {
	lv := NewLogView()
	lv.SetHighlightCurrentEvent(true)

	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	result := lv.ScrollToEventID("e20")

	if !result || lv.GetCurrentEvent().EventID != "e20" {
		t.Errorf("Scrolling to event id failed. result=%t, eventID=%s", result, lv.GetCurrentEvent().EventID)
	}
}

func TestLogView_ScrollToTimestamp(t *testing.T) {
	lv := NewLogView()
	lv.SetHighlightCurrentEvent(true)

	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	ts = ts.Add(20 * time.Second)

	result := lv.ScrollToTimestamp(ts)

	if !result || lv.GetCurrentEvent().EventID != "e20" {
		t.Errorf("Scrolling to event id failed. result=%t, eventID=%s, eventTime=%v", result, lv.GetCurrentEvent().EventID, lv.GetCurrentEvent().Timestamp)
	}
}

func TestLogView_ScrollToTop(t *testing.T) {
	lv := NewLogView()
	lv.SetHighlightCurrentEvent(true)
	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	lv.ScrollToTop()

	if lv.GetCurrentEvent().EventID != "e0" {
		t.Errorf("Failed to scroll to top")
	}
}

func TestLogView_ScrollToBottom(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	screen.SetSize(100, 10)
	lv := NewLogView()
	lv.SetHighlightCurrentEvent(true)
	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))
	lv.Draw(screen) // prime page sizes

	lv.ScrollToTop()
	lv.SetFollowing(false)
	lv.ScrollToBottom()

	if lv.GetCurrentEvent().EventID != "e99" ||
		lv.top.EventID != "e90" ||
		!lv.IsFollowing() {
		t.Errorf("Failed to scroll to top, current eventID=%s, top EventID=%s, following=%t",
			lv.GetCurrentEvent().EventID, lv.top.EventID, lv.IsFollowing())
	}
}

func TestLogView_SetMaxEvents(t *testing.T) {
	lv := NewLogView()
	lv.SetMaxEvents(10)
	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	if lv.firstEvent.EventID != "e90" && lv.EventCount() != 10 {
		t.Errorf("Failed to limit max events, eventCount=%d, should be 10", lv.EventCount())
	}
}

func TestLogView_SetMaxEventsAfter(t *testing.T) {
	lv := NewLogView()
	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	lv.SetMaxEvents(10)

	if lv.firstEvent.EventID != "e90" && lv.EventCount() != 10 {
		t.Errorf("Failed to limit max events, eventCount=%d, should be 10", lv.EventCount())
	}
}

func TestLogView_Highlighting(t *testing.T) {
	lv := NewLogView()
	lv.SetHighlightPattern(`(?P<ts>\d{2}:\d{2}:\d{2}.\d{3})\s+\[(?P<thread>.*)\]\s+(?P<level>\S+)\s+(?P<class>[a-zA-Z0-9_.]+).*(?:in (?P<elapsed>\d+)ms)?`)
	ts := time.Now().Add(-24 * time.Hour)
	lv.AppendEvents(randomEvents(100, ts))

	lv.SetMaxEvents(10)

	if lv.firstEvent.EventID != "e90" && lv.EventCount() != 10 {
		t.Errorf("Failed to limit max events, eventCount=%d, should be 10", lv.EventCount())
	}
}

func TestLogView_ConcatenateEvents(t *testing.T) {
	lv := NewLogView()
	lv.SetConcatenateEvents(true)
	lv.SetNewEventMatchingRegex(`^[^\s]`)

	event1 := NewLogEvent("1", "Line 1")
	event2 := NewLogEvent("2", " and still line 1")

	lv.AppendEvent(event1)
	lv.AppendEvent(event2)

	if lv.EventCount() != 1 {
		t.Errorf("EventCount must be 1")
	}

	if lv.GetCurrentEvent().EventID != "1" && lv.GetCurrentEvent().Message != "Line 1 and still line 1" {
		t.Errorf("Excpected eventID=1 and concatenated event line")
	}

	if lv.current != lv.firstEvent && lv.current != lv.lastEvent {
		t.Errorf("current event must be the first and the last")
	}
}

func TestLogView_WrapEvent(t *testing.T) {
	lv := NewLogView()
	lv.pageWidth = 20
	lv.AppendEvent(NewLogEvent("1", "Test 1"))
	lv.AppendEvent(NewLogEvent("2", "Test 2"))
	lv.SetLineWrap(true)

	//										   1        10        20       30
	//                                         |        |         |        |
	event := NewLogEvent("3", "Line is wide але it has a\nnew line")
	lv.AppendEvent(event)

	if lv.EventCount() != 3 {
		t.Errorf("EventCount must be 3")
	}

	e := lv.firstEvent.next.next
	if string(e.Runes[e.start:e.end]) != "Line is wide але it " || e.lineCount != 3 || e.order != 1 {
		t.Errorf("Invalid first line")
	}
	e = e.next
	if string(e.Runes[e.start:e.end]) != "has a\n" || e.order != 2 {
		t.Errorf("Invalid second line")
	}
	e = e.next
	if string(e.Runes[e.start:e.end]) != "new line" || e.order != 3 {
		t.Errorf("Invalid third line")
	}
}

func TestLogView_replaceEvents(t *testing.T) {
	lv := NewLogView()
	lv.AppendEvent(NewLogEvent("1", "Test 1"))
	lv.AppendEvent(NewLogEvent("2", "Test 2"))
	lv.AppendEvent(NewLogEvent("3", "Test 3"))

	lv.current = lv.firstEvent.next

	toReplace := lv.firstEvent.next

	ne := lv.replaceEvent(toReplace, []*logEventLine{
		{
			EventID:   "2",
			order:     1,
			lineCount: 2,
			Runes:     []rune("Test2.1"),
		},
		{
			EventID:   "2",
			order:     2,
			lineCount: 2,
			Runes:     []rune("Test2.2"),
		},
	})

	if lv.eventCount != 3 {
		t.Errorf("Should not change event count")
	}
	ne = ne.previous
	if ne.order != 1 && string(ne.Runes) != "Test2.1" && ne.previous != lv.firstEvent {
		t.Errorf("Invalid first replacement event")
	}
	if lv.current != ne {
		t.Errorf("Current event must point to the new replacement")
	}
	if lv.firstEvent.next != ne {
		t.Errorf("First event's next must point to the new replacement")
	}
	ne = ne.next
	if ne.order != 2 && string(ne.Runes) != "Test2.2" && ne.next != lv.lastEvent {
		t.Errorf("Invalid first replacement event")
	}
	if lv.lastEvent.previous != ne {
		t.Errorf("Last event's previous must point to the new replacement")
	}
}

func TestLogView_replaceLastEvent(t *testing.T) {
	lv := NewLogView()
	lv.AppendEvent(NewLogEvent("1", "Test 1"))
	lv.AppendEvent(NewLogEvent("2", "Test 2"))

	lv.current = lv.firstEvent.next

	toReplace := lv.firstEvent.next

	ne := lv.replaceEvent(toReplace, []*logEventLine{
		{
			EventID:   "2",
			order:     1,
			lineCount: 2,
			Runes:     []rune("Test2.1"),
		},
		{
			EventID:   "2",
			order:     2,
			lineCount: 2,
			Runes:     []rune("Test2.2"),
		},
	})

	if lv.eventCount != 2 {
		t.Errorf("Should not change event count")
	}
	ne = ne.previous
	if ne.order != 1 && string(ne.Runes) != "Test2.1" && ne.previous != lv.firstEvent {
		t.Errorf("Invalid first replacement event")
	}
	if lv.current != ne {
		t.Errorf("Current event must point to the new replacement")
	}
	if lv.firstEvent.next != ne {
		t.Errorf("First event's next must point to the new replacement")
	}
	ne = ne.next
	if ne.order != 2 && string(ne.Runes) != "Test2.2" && ne.next != nil {
		t.Errorf("Invalid first replacement event")
	}
	if lv.lastEvent != ne {
		t.Errorf("Last event must point to the new replacement")
	}
}

func TestLogView_replaceFirstEvent(t *testing.T) {
	lv := NewLogView()
	lv.AppendEvent(NewLogEvent("2", "Test 2"))
	lv.AppendEvent(NewLogEvent("3", "Test 3"))

	lv.current = lv.firstEvent.next

	toReplace := lv.firstEvent.next

	ne := lv.replaceEvent(toReplace, []*logEventLine{
		{
			EventID:   "2",
			order:     1,
			lineCount: 2,
			Runes:     []rune("Test2.1"),
		},
		{
			EventID:   "2",
			order:     2,
			lineCount: 2,
			Runes:     []rune("Test2.2"),
		},
	})

	if lv.eventCount != 2 {
		t.Errorf("Should not change event count")
	}
	ne = ne.previous
	if ne.order != 1 && string(ne.Runes) != "Test2.1" && ne.previous != nil {
		t.Errorf("Invalid first replacement event")
	}
	if lv.current != ne {
		t.Errorf("Current event must point to the new replacement")
	}
	if lv.firstEvent.next != ne {
		t.Errorf("First event must point to the new replacement")
	}
	ne = ne.next
	if ne.order != 2 && string(ne.Runes) != "Test2.2" && ne.next != lv.lastEvent {
		t.Errorf("Invalid first replacement event")
	}
	if lv.lastEvent != ne {
		t.Errorf("Last event's previous must point to the new replacement")
	}
}

func TestLogView_mergeWrappedLines(t *testing.T) {
	lv := NewLogView()
	lv.pageWidth = 20
	lv.AppendEvent(NewLogEvent("1", "This is a rather long event and it should be wrapped"))

	if lv.firstEvent.lineCount != 3 {
		t.Errorf("Event should wrap multiple lines, but got: %d", lv.firstEvent.lineCount)
	}
	e := lv.firstEvent.next
	e1 := lv.mergeWrappedLines(e)
	if string(e1.Runes) != "This is a rather long event and it should be wrapped" {
		t.Errorf("Invalid text in unwrapped event")
	}
	if lv.firstEvent != e1 && lv.lastEvent != e1 {
		t.Errorf("first and last should point to unwrapped event")
	}
}

// A wrapped event that is the last one in the list and consists of a single piece
// (e.g. a short line ending with a new line) must survive unwrapping, as happens on resize.
func TestLogView_mergeWrappedLinesSinglePieceLast(t *testing.T) {
	lv := NewLogView()
	lv.pageWidth = 20
	lv.SetLineWrap(true)
	lv.AppendEvent(NewLogEvent("1", "short\n"))

	if lv.firstEvent.order != 1 || lv.firstEvent.next != nil {
		t.Fatalf("expected a single wrapped piece, got order=%d", lv.firstEvent.order)
	}
	e := lv.mergeWrappedLines(lv.firstEvent)
	if e.order != 0 || e.next != nil || lv.lastEvent != e {
		t.Errorf("event must be unwrapped and remain the last one")
	}
}

// Merging a single-piece wrapped event must not swallow the event that follows it.
func TestLogView_mergeWrappedLinesSinglePieceKeepsNext(t *testing.T) {
	lv := NewLogView()
	lv.pageWidth = 20
	lv.SetLineWrap(true)
	lv.AppendEvent(NewLogEvent("1", "short\n"))
	lv.AppendEvent(NewLogEvent("2", "next event"))

	lv.mergeWrappedLines(lv.firstEvent)
	if lv.firstEvent.next == nil || lv.firstEvent.next.EventID != "2" || lv.firstEvent.next.previous != lv.firstEvent {
		t.Errorf("following event was lost or mislinked")
	}
}

// Repeated rewrap with changing width (terminal resize) must not panic or lose events.
func TestLogView_rewrapOnResize(t *testing.T) {
	lv := NewLogView()
	lv.SetLineWrap(true)
	lv.pageWidth = 20
	lv.AppendEvent(NewLogEvent("1", "short\n"))
	lv.AppendEvent(NewLogEvent("2", "This is a rather long event and it should be wrapped"))
	lv.AppendEvent(NewLogEvent("3", "tail\n"))

	for _, w := range []int{10, 80, 5, 20, 200, 7} {
		lv.pageWidth = w
		lv.rewrapLines()
		ids := map[string]bool{}
		for e := lv.firstEvent; e != nil; e = e.next {
			ids[e.EventID] = true
		}
		if len(ids) != 3 {
			t.Fatalf("width %d: events lost, got %v", w, ids)
		}
	}
}

func TestLogView_FindMatchingEvent(t *testing.T) {
	lv := NewLogView()
	lv.AppendEvents(randomEvents(100, time.Now()))

	event := lv.FindMatchingEvent("e20", func(event *LogEvent) bool {
		return event.EventID == "e50"
	})

	if event == nil || event.EventID != "e50" {
		t.Errorf("Failed to find event with id == e50, event: %v", event)
	}

	event = lv.FindMatchingEvent("e50", func(event *LogEvent) bool {
		return event.EventID == "e20"
	})

	if event != nil {
		t.Errorf("Found event with id == e20, should have returned nil")
	}
}

func BenchmarkLogView(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	lv := NewLogView()
	lv.SetHighlightPattern(`(?P<g1>Event)\s+(?P<g2>#\d+)\s+(?P<g3>.*)`)
	lv.SetHighlighting(true)
	lv.SetLevelHighlighting(true)

	ts := time.Now().Add(-24 * time.Hour)
	events := randomBenchEvents(eventCount, ts)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		lv.AppendEvents(events)
		lv.Draw(screen)
	}

}

func randomBenchEvents(count int, startingTimestamp time.Time) []*LogEvent {
	result := make([]*LogEvent, count)
	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)
		logEvent := NewLogEvent("e"+idx, "Event #"+idx+" !")
		logEvent.Timestamp = startingTimestamp.Add(time.Duration(i) * time.Second)
		result[i] = logEvent
	}
	return result
}

func randomEvents(count int, startingTimestamp time.Time) []*LogEvent {
	result := make([]*LogEvent, count)
	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)
		logEvent := NewLogEvent("e"+idx, "Event #"+idx)
		logEvent.Timestamp = startingTimestamp.Add(time.Duration(i) * time.Second)
		result[i] = logEvent
	}
	return result
}
