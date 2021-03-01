package main

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

func BenchmarkLogView(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	lv := NewLogView()
	lv.SetHighlightPattern(`(?P<g1>Event)\s+(?P<g2>#\d+)\s+(?P<g3>.*)`)
	lv.SetHighlightColorFg("g1", tcell.ColorDarkCyan)
	lv.SetHighlightColorFg("g2", tcell.ColorDarkGreen)
	lv.SetHighlightColorFg("g3", tcell.ColorYellow)
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
