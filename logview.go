package logview

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/gdamore/tcell/v2"
	gui "github.com/rivo/tview"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type styledSpan struct {
	start int
	end   int
	style tcell.Style
}

type logEventLine struct {
	EventID    string
	Source     string
	Timestamp  time.Time
	Level      LogLevel
	Runes      []rune
	lineID     uint
	previous   *logEventLine
	next       *logEventLine
	styleSpans []styledSpan

	// start and end determine slice of LogEvent.Message this event line covers
	// for unwrapped string this will be the whole length of the message starting at position 0
	// for wrapped strings each line with order != 0 will cover its portion of main event message
	start int
	end   int

	// order indicate whether the single log event is split over multiple lines because of wrapping
	// if the event is not split, then order will be 0
	// otherwise the first line will have the order value of 1, the next line is 2 and so on
	order     int
	lineCount uint
	// there are newline symbols in the message. Normally there shouldn't be any, but if we've done
	// event merging, then merged parts will be separated by newlines. We need to know if there are any
	// so we can decide if we need to wrap them.
	hasNewLines bool
}

func (e logEventLine) AsLogEvent() *LogEvent {
	return &LogEvent{
		EventID:   e.EventID,
		Source:    e.Source,
		Timestamp: e.Timestamp,
		Level:     e.Level,
		Message:   string(e.Runes),
	}
}

func (e logEventLine) message() string {
	return string(e.Runes)
}

func (e logEventLine) getLineCount() uint {
	return e.lineCount
}

func (e *logEventLine) copy() *logEventLine {
	eventCopy := &logEventLine{
		EventID:     e.EventID,
		Source:      e.Source,
		Timestamp:   e.Timestamp,
		Level:       e.Level,
		Runes:       e.Runes,
		lineID:      e.lineID,
		previous:    e.previous,
		next:        e.next,
		styleSpans:  e.styleSpans,
		start:       e.start,
		end:         e.end,
		order:       e.order,
		lineCount:   e.lineCount,
		hasNewLines: e.hasNewLines,
	}
	return eventCopy
}

// OnCurrentChanged is an event time that is fired when current log event is changed
type OnCurrentChanged func(current *LogEvent)

// LogView is a Box that displays log events
//
// LogView doesn't have border or scroll bars to allow easier copy-paste of events.
type LogView struct {
	*gui.Box

	firstEvent *logEventLine
	lastEvent  *logEventLine
	top        *logEventLine
	current    *logEventLine
	eventCount uint
	eventLimit uint

	newEventMatcher   *regexp.Regexp
	concatenateEvents bool

	highlightingEnabled bool
	highlightPattern    *regexp2.Regexp

	highlightLevels bool
	warningBgColor  tcell.Color
	errorBgColor    tcell.Color

	highlightCurrent bool
	currentBgColor   tcell.Color

	sourceStyle    tcell.Style
	timestampStyle tcell.Style

	// as new events are appended, older events are scrolled up, like tail -f
	following bool

	showSource       bool
	sourceClipLength int
	showTimestamp    bool
	timestampFormat  string
	wrap             bool

	defaultStyle tcell.Style

	hasFocus bool

	lastWidth, lastHeight int
	pageHeight, pageWidth int
	fullPageWidth         int
	screenCoords          []int

	onCurrentChanged OnCurrentChanged

	// force re-wrapping on next draw
	forceWrap bool

	sync.RWMutex
}

// NewLogView returns a new log view.
func NewLogView() *LogView {
	defaultStyle := tcell.StyleDefault.Foreground(gui.Styles.PrimaryTextColor).Background(gui.Styles.PrimitiveBackgroundColor)
	logView := &LogView{
		Box:                 gui.NewBox(),
		showSource:          false,
		showTimestamp:       false,
		timestampFormat:     "15:04:05.000",
		sourceClipLength:    6,
		wrap:                true,
		following:           true,
		highlightingEnabled: true,
		defaultStyle:        defaultStyle,
		currentBgColor:      tcell.ColorDimGray,
		warningBgColor:      tcell.ColorSaddleBrown,
		errorBgColor:        tcell.ColorIndianRed,
		sourceStyle:         defaultStyle.Foreground(tcell.ColorDarkGoldenrod),
		timestampStyle:      defaultStyle.Foreground(tcell.ColorDarkOrange),
		screenCoords:        make([]int, 2),
		concatenateEvents:   false,
		newEventMatcher:     regexp.MustCompile(`^[^\s]`),
	}
	logView.Box.SetBorder(false)
	return logView
}

// GetWidth returns the width of the list view
func (lv *LogView) GetWidth() int {
	lv.RLock()
	defer lv.RUnlock()

	return lv.fullPageWidth
}

// GetHeight returns the width of the list view
func (lv *LogView) GetHeight() int {
	lv.RLock()
	defer lv.RUnlock()

	return lv.pageHeight
}

// Clear deletes all events from the log view
func (lv *LogView) Clear() {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	lv.firstEvent = nil
	lv.lastEvent = nil
	lv.current = nil
	lv.top = nil
	lv.eventCount = 0
}

// GetEventCount returns number of events in the log view
func (lv *LogView) GetEventCount() uint {
	lv.RLock()
	defer lv.RUnlock()
	return lv.eventCount
}

// SetMaxEvents sets a maximum number of events that log view will hold
func (lv *LogView) SetMaxEvents(limit uint) {
	lv.Lock()
	defer lv.Unlock()

	lv.eventLimit = limit
	lv.ensureEventLimit()
}

// GetMaxEvents returns a maximum number of events that log view will hold
func (lv *LogView) GetMaxEvents() uint {
	lv.RLock()
	defer lv.RUnlock()

	return lv.eventLimit
}

// SetLineWrap enables/disables the line wrap. Disabling line wrap may increase performance
func (lv *LogView) SetLineWrap(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	if lv.wrap != enabled {
		lv.forceWrap = true
	}
	lv.wrap = enabled
}

// IsLineWrapEnabled returns the current status of line wrap
func (lv *LogView) IsLineWrapEnabled() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.wrap
}

// SetConcatenateEvents enables/disables event concatenation
//
// Events with a message that do not match regular expression set by SetNewEventMatcher are appended to the previous
// event
func (lv *LogView) SetConcatenateEvents(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.concatenateEvents = enabled
}

// IsConcatenateEventsEnabled returns the status of event concatenation
func (lv *LogView) IsConcatenateEventsEnabled() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.concatenateEvents
}

// SetNewEventMatcher sets the regular expression to use for detecting continuation events.
//
// If event message matches provided regular expression it is treated as a new event, otherwise it is appended to
// a previous line. All attributes of appended event are discarded.
//
// If line wrapping is enabled, event will be split into original lines
//
// For example typical Java exception looks like
// IllegalArgumentException: No nulls please
//	   caused by NullPointerException
//     at org.some.java.package.Class
//     at org.another.java.package
//
// Each line might be a new log event, but it makes sense to treat whole stack trace as as a single event.
func (lv *LogView) SetNewEventMatchingRegex(regex string) {
	lv.Lock()
	defer lv.Unlock()

	if regex == "" {
		lv.newEventMatcher = nil
	} else {
		lv.newEventMatcher = regexp.MustCompile(regex)
	}
}

// GetNewEventMatchingRegex gets the regular expression used for detecting continuation events.
func (lv *LogView) GetNewEventMatchingRegex() string {
	lv.RLock()
	defer lv.RUnlock()

	if lv.newEventMatcher == nil {
		return ""
	} else {
		return lv.newEventMatcher.String()
	}
}

// SetTextStyle sets the default style for the log messages
func (lv *LogView) SetTextStyle(style tcell.Style) {
	lv.Lock()
	defer lv.Unlock()

	lv.defaultStyle = style
}

// SetSourceStyle sets the style for displaying event source
func (lv *LogView) SetSourceStyle(style tcell.Style) {
	lv.Lock()
	defer lv.Unlock()

	lv.sourceStyle = style
}

// SetTimestampStyle sets the style for displaying event timestamp
func (lv *LogView) SetTimestampStyle(style tcell.Style) {
	lv.Lock()
	defer lv.Unlock()

	lv.timestampStyle = style
}

// SetCurrentBgColor sets the background color to highlight currently selected event
func (lv *LogView) SetCurrentBgColor(color tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.currentBgColor = color
}

// SetEventLimit sets the limit to number of log event held by log view.
//
// To disable limit set it to zero.
func (lv *LogView) SetEventLimit(limit uint) {
	lv.Lock()
	defer lv.Unlock()

	lv.eventLimit = limit

	lv.ensureEventLimit()
}

// RefreshHighlights forces recalculation of highlight patterns for all events in the log view.
// LogView calculates highlight spans once for each event when the event is appended. Any changes in highlighting
// will not be applied to the events that are already in the log view.
// To apply changes to all the events call this function.
// Warning: this might be a rather expensive operation
func (lv *LogView) RefreshHighlights() {
	lv.Lock()
	defer lv.Unlock()

	lv.recolorizeLines()
}

// SetHighlightPattern sets new regular expression pattern to find spans that need to be highlighted
// setting this to nil disables highlighting.
//
// pattern is a regular expression where each matching named capturing group can be highlighted with a different color.
// Colors for any given group name can be set with SetHighlightColor, SetHighlightColorFg, SetHighlightColorBg
//
// Note:
// Updating pattern doesn't changes highlighting for previously appended events
// Call RefreshHighlights() to force updating highlighting for all events in the log view.
func (lv *LogView) SetHighlightPattern(pattern string) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightPattern = regexp2.MustCompile(pattern, regexp2.IgnoreCase+regexp2.RE2)
}

// SetHighlighting enables/disables event message highlighting according to the pattern set by SetHighlightPattern.
//
// Events appended when this setting was disabled will not be highlighted until RefreshHighlights function is called.
func (lv *LogView) SetHighlighting(enable bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightingEnabled = enable
}

// IsHighlightingEnabled returns the status of event message highlighting
func (lv *LogView) IsHighlightingEnabled() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.highlightingEnabled
}

// SetWarningColor sets the background color for events with level == LogLevelWarning.
// Event level highlighting can be turned on and off with SetLevelHighlighting function.
//
// Changing warning color will do nothing to the events that are already in the log view. To update
// highlighting of all events use RefreshHighlights. Be warned: this is an expensive operation
func (lv *LogView) SetWarningBgColor(bgColor tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.warningBgColor = bgColor
}

// SetErrorColor sets the background color for events with level == LogLevelError.
// Event level highlighting can be turned on and off with SetLevelHighlighting function.
//
// Changing error color will do nothing to the events that are already in the log view. To update
// highlighting of all events use RefreshHighlights. Be warned: this is an expensive operation
func (lv *LogView) SetErrorBgColor(bgColor tcell.Color) {
	lv.Lock()
	defer lv.Unlock()

	lv.errorBgColor = bgColor
}

// SetLevelHighlighting enables background color highlighting for events based on severity level
func (lv *LogView) SetLevelHighlighting(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightLevels = enabled
}

// IsLevelHighlightingEnabled returns the status of background color highlighting for events based on severity level
func (lv *LogView) IsLevelHighlightingEnabled() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.highlightLevels
}

// SetHighlightCurrentEvent enables background color highlighting for currently selected event
func (lv *LogView) SetHighlightCurrentEvent(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.highlightCurrent = enabled
}

// IsHighlightCurrentEventEnabled returns the status background color highlighting for currently selected event
func (lv *LogView) IsHighlightCurrentEventEnabled() bool {
	lv.Lock()
	defer lv.Unlock()

	return lv.highlightCurrent
}

// GetCurrentEvent returns the currently selected event
func (lv *LogView) GetCurrentEvent() *LogEvent {
	lv.RLock()
	defer lv.RUnlock()

	return lv.current.AsLogEvent()
}

// FindMatchingEvent searches for a event that matches a given predicate
// First event to be matched is the one with event id equal to startingEventId. If the startingEventId is an empty
// string, the search will start from the first event in the log view.
//
// If no such event can be found it will return nil
func (lv *LogView) FindMatchingEvent(startingEventId string, predicate func(event *LogEvent) bool) *LogEvent {
	lv.Lock()
	defer lv.Unlock()

	event := lv.findByEventId(startingEventId)
	for event != nil {
		logEvent := event.AsLogEvent()
		if predicate(logEvent) {
			return logEvent
		}
		event = event.next
	}
	return nil
}

// GetFirstEvent returns the first event in the log view
func (lv *LogView) GetFirstEvent() *LogEvent {
	lv.Lock()
	defer lv.Unlock()

	return lv.firstEvent.AsLogEvent()
}

// SetBorder does nothing
func (lv *LogView) SetBorder(_ bool) {
	// do nothing
}

// Focus is called when this primitive receives focus.
func (lv *LogView) Focus(_ func(p gui.Primitive)) {
	lv.Lock()
	defer lv.Unlock()

	// Implemented here with locking because this is used by layout primitives.
	lv.hasFocus = true
}

// HasFocus returns whether or not this primitive has focus.
func (lv *LogView) HasFocus() bool {
	lv.RLock()
	defer lv.RUnlock()

	// Implemented here with locking because this may be used in the "changed"
	// callback.
	return lv.hasFocus
}

// Draw draws this primitive onto the screen.
func (lv *LogView) Draw(screen tcell.Screen) {
	//if !lv.GetVisible() {
	//	return
	//}

	lv.Box.Draw(screen)

	lv.Lock()
	defer lv.Unlock()

	// Get the available size.
	x, y, width, height := lv.GetInnerRect()
	if height == 0 {
		return
	}
	lv.screenCoords[0] = x
	lv.screenCoords[1] = y

	lv.fullPageWidth = width
	lv.pageHeight = height
	lv.pageWidth = width
	if lv.isHeaderPossible() {
		lv.pageWidth -= lv.headerWidth()
	}
	if (width != lv.lastWidth || height != lv.lastHeight && lv.wrap) || lv.forceWrap {
		lv.forceWrap = false
		lv.rewrapLines()
		if lv.following {
			// ensure correct top line when resizing
			lv.scrollToEnd()
		}
	}
	lv.lastWidth, lv.lastHeight = width, height

	line := y

	top := lv.top
	for top != nil && line < y+height {
		lv.drawEvent(screen, x, line, top)
		line++
		top = top.next
	}
	for line < y+height {
		lv.clearLine(screen, x, line)
		line++
	}
}

// AppendEvent appends an event to the log view
// If possible use AppendEvents to add multiple events at once
func (lv *LogView) AppendEvent(logEvent *LogEvent) {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	lv.append(logEvent)
}

// AppendEvents appends multiple events in a single batch improving performance
func (lv *LogView) AppendEvents(events []*LogEvent) {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	for _, e := range events {
		lv.append(e)
	}
}

// ScrollPageDown scrolls the log view one screen down
//
// This will enable auto follow if the last line has been reached
func (lv *LogView) ScrollPageDown() {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	lv.scrollPageDown()
}

// ScrollPageUp scrolls the log view one screen up
//
// This does not disables following.
func (lv *LogView) ScrollPageUp() {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	lv.scrollPageUp()
}

// ScrollToBottom scrolls the log view to the last event
//
// This does not automatically enables following. User SetFollowing function to enable it
func (lv *LogView) ScrollToBottom() {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	lv.scrollToEnd()
}

// ScrollToTop scrolls the log view to the first event
//
// This does not automatically enables following. User SetFollowing function to enable it
func (lv *LogView) ScrollToTop() {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	lv.scrollToStart()
}

// SetFollowing enables/disables following mode. Following mode automatically scrolls log view up
// as new events are appended. Last event is always in the view
//
// Enabling following will automatically scroll to the last event
func (lv *LogView) SetFollowing(follow bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.following = follow
	if follow {
		lv.scrollToEnd()
	}
}

// EventCount returns the number of log events in the log view
func (lv *LogView) EventCount() uint {
	lv.RLock()
	defer lv.RUnlock()

	return lv.eventCount
}

// IsFollowing returns whether the following mode is enabled. Following mode automatically scrolls log view up
// as new events are appended. Last event is always in the view.
func (lv *LogView) IsFollowing() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.following
}

// ScrollToTimestamp scrolls to the first event with a timestamp equal to or greater than given.
// If no event satisfies that condition it will not scroll and return false.
//
// Current event will be updated to the found event
func (lv *LogView) ScrollToTimestamp(timestamp time.Time) bool {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	event := lv.firstEvent
	for event != nil && event.Timestamp.Before(timestamp) {
		event = event.next
	}
	if event == nil {
		return false
	}
	lv.top = event
	lv.current = event
	lv.adjustTop()
	for i := 0; i < lv.pageHeight/4; i++ { // scroll a little bit back
		if lv.top.previous != nil {
			lv.top = lv.top.previous
		}
	}
	return true
}

// ScrollToEventID scrolls to the first event with a matching eventID
// If no such event is found it will not scroll and return false.
//
// Current event will be updated to the found event
func (lv *LogView) ScrollToEventID(eventID string) bool {
	defer lv.fireOnCurrentChange(lv.current)
	lv.Lock()
	defer lv.Unlock()

	event := lv.findByEventId(eventID)
	if event == nil {
		return false
	}
	lv.top = event
	lv.current = event
	lv.adjustTop()
	for i := 0; i < lv.pageHeight/4; i++ {
		if lv.top.previous != nil {
			lv.top = lv.top.previous
		}
	}
	return true
}

// SetShowSource enables/disables the displaying of event source
//
// Event Source is displayed to the left of the actual event message with style defined by SetSourceStyle and
// is clipped to the length set by SetSourceClipLength (6 characters is the default)
func (lv *LogView) SetShowSource(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.showSource = enabled
}

// IsShowSource returns whether the showing of event source is enabled
func (lv *LogView) IsShowSource() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.showSource
}

// SetSourceClipLength sets the maximum length of event source that would be displayed if SetShowSource is on
func (lv *LogView) SetSourceClipLength(length int) {
	lv.Lock()
	defer lv.Unlock()

	lv.sourceClipLength = length
}

// GetSourceClipLength returns the current maximum length of event source that would be displayed
func (lv *LogView) GetSourceClipLength() int {
	lv.RLock()
	defer lv.RUnlock()

	return lv.sourceClipLength
}

// SetShowTimestamp enables/disables the displaying of event timestamp
//
// Event timestamp is displayed to the left of the actual event message with the format defined by SetTimestampFormat
func (lv *LogView) SetShowTimestamp(enabled bool) {
	lv.Lock()
	defer lv.Unlock()

	lv.showTimestamp = enabled
}

// IsShowTimestamp returns whether the showing of event source is enabled
func (lv *LogView) IsShowTimestamp() bool {
	lv.RLock()
	defer lv.RUnlock()

	return lv.showTimestamp
}

// SetTimestampFormat sets the format for displaying the event timestamp.
//
// Default is 15:04:05.000
func (lv *LogView) SetTimestampFormat(format string) {
	lv.Lock()
	defer lv.Unlock()

	lv.timestampFormat = format
}

// GetTimestampFormat returns the format used to display timestamps
func (lv *LogView) GetTimestampFormat() string {
	lv.RLock()
	defer lv.RUnlock()

	return lv.timestampFormat
}

// InputHandler returns the handler for this primitive.
func (lv *LogView) InputHandler() func(event *tcell.EventKey, setFocus func(p gui.Primitive)) {
	return lv.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p gui.Primitive)) {
		defer lv.fireOnCurrentChange(lv.current)
		lv.Lock()
		defer lv.Unlock()

		if HitShortcut(event, Keys.MoveFirst, Keys.MoveFirst2) {
			lv.scrollToStart()
		} else if HitShortcut(event, Keys.MoveLast, Keys.MoveLast2) {
			lv.scrollToEnd()
		} else if HitShortcut(event, Keys.MoveUp, Keys.MoveUp2) {
			lv.scrollOneUp()
		} else if HitShortcut(event, Keys.MoveDown, Keys.MoveDown2) {
			lv.scrollOneDown()
		} else if HitShortcut(event, Keys.MovePreviousPage) {
			lv.scrollPageUp()
		} else if HitShortcut(event, Keys.MoveNextPage) {
			lv.scrollPageDown()
		}
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (lv *LogView) MouseHandler() func(action gui.MouseAction, event *tcell.EventMouse, setFocus func(p gui.Primitive)) (consumed bool, capture gui.Primitive) {
	return lv.WrapMouseHandler(func(action gui.MouseAction, event *tcell.EventMouse, setFocus func(p gui.Primitive)) (consumed bool, capture gui.Primitive) {
		x, y := event.Position()
		if !lv.InRect(x, y) {
			return false, nil
		}

		switch action {
		case gui.MouseLeftClick:
			defer lv.fireOnCurrentChange(lv.current)
			consumed = true
			setFocus(lv)
			lv.Lock()
			localY := y - lv.screenCoords[1]
			lv.current = lv.atOffset(lv.top, localY)
			lv.Unlock()
			if lv.onCurrentChanged != nil {
				lv.onCurrentChanged(lv.current.AsLogEvent())
			}
		case gui.MouseScrollUp:
			lv.ScrollPageUp()
			consumed = true
		case gui.MouseScrollDown:
			lv.ScrollPageDown()
			consumed = true
		}

		return
	})
}

// SetOnCurrentChange sets a listener that will be called every time the current event is changed
//
// If current event highlighting is disabled, listener will not be called.
func (lv *LogView) SetOnCurrentChange(listener OnCurrentChanged) {
	lv.Lock()
	defer lv.Unlock()

	lv.onCurrentChanged = listener
}

// *******************************
// internal implementation details

func (lv *LogView) fireOnCurrentChange(oldCurrent *logEventLine) {
	if oldCurrent != lv.current && lv.onCurrentChanged != nil && lv.highlightCurrent {
		lv.onCurrentChanged(lv.current.AsLogEvent())
	}
}

func (lv *LogView) append(logEvent *LogEvent) {
	var event *logEventLine

	if !lv.concatenateEvents || lv.newEventMatcher == nil || lv.newEventMatcher.MatchString(logEvent.Message) || lv.lastEvent == nil {
		// defensive copy of Log event
		event = &logEventLine{
			EventID:     logEvent.EventID,
			Source:      logEvent.Source,
			Timestamp:   logEvent.Timestamp,
			Level:       logEvent.Level,
			Runes:       []rune(logEvent.Message),
			lineCount:   1,
			lineID:      lv.eventCount + 1,
			start:       0,
			order:       0,
			end:         utf8.RuneCountInString(logEvent.Message),
			hasNewLines: strings.Contains(logEvent.Message, "\n"),
		}
		lv.insertAfter(lv.lastEvent, event, true)
	} else {
		event = lv.lastEvent
		event.Runes = append(event.Runes, []rune("\n"+logEvent.Message)...)
		event.hasNewLines = event.hasNewLines || strings.Contains(logEvent.Message, "\n")
		event = lv.mergeWrappedLines(event)
	}

	// process event
	lv.colorize(event)
	lv.calculateWrap(event)

	lv.ensureEventLimit()

	// if we're in following mode and have enough events to fill the page then update the top position
	if lv.following && lv.eventCount > uint(lv.pageHeight) {
		lv.top = lv.atOffset(lv.lastEvent, -lv.pageHeight)
		lv.current = event
	}
}

// atOffset finds event that is at given offset from the starting event
// offset can be positive or negative
// if first or last event is reached then it is returned
func (lv *LogView) atOffset(start *logEventLine, offset int) *logEventLine {
	if offset == 0 {
		return start
	}
	current := start
	var steps int
	if offset > 0 {
		steps = offset
	} else {
		steps = -offset
	}
	for steps > 0 {
		if (current == lv.firstEvent && offset < 0) || (current == lv.lastEvent && offset > 0) {
			break
		}
		if offset < 0 {
			current = current.previous
		} else {
			current = current.next
		}
		steps--
	}
	return current
}

// calculateWrap splits the event line into multiple according to the wrap flag and window width
// for every split event it deletes previous wrapped lines and calculates wrapping from scratch
// new event lines with order >= 1 are created and inserted in the log list
// last event is returned
func (lv *LogView) calculateWrap(event *logEventLine) *logEventLine {
	if !lv.wrap || lv.pageWidth == 0 || (len(event.Runes) <= lv.pageWidth && !event.hasNewLines) {
		if event.order != 0 { // no wrapping needed, but the line is wrapped
			event = lv.mergeWrappedLines(event)
		}
		return event
	}
	if event.order != 0 { // first drop extra event lines
		event = lv.mergeWrappedLines(event)
	}

	lineLength := len(event.Runes)
	start := 0
	end := 0
	events := make([]*logEventLine, 0)

	for end < lineLength {
		if end-start == lv.pageWidth || event.Runes[end] == '\n' { // wrap here
			currentEvent := event.copy()
			currentEvent.hasNewLines = false
			currentEvent.start = start
			if event.Runes[end] == '\n' {
				end = end + 1
				currentEvent.hasNewLines = true
			}
			currentEvent.end = end

			start = end
			events = append(events, currentEvent)
		} else {
			end++
		}
	}
	if end > start { // add final piece
		currentEvent := event.copy()
		currentEvent.start = start
		currentEvent.end = end
		events = append(events, currentEvent)
	}
	for i, r := range events {
		r.order = i + 1
		r.lineCount = uint(len(events))
	}
	return lv.replaceEvent(event, events)
}

func findFirstWrappedLine(event *logEventLine) *logEventLine {
	for event.order > 1 && event.previous != nil {
		event = event.previous
	}
	return event
}

// mergeWrappedLines will delete all extra lines for an event
// if the event order is == 0, it will return the event
// otherwise it will fine the first event, change its order to 0
// and delete all subsequent events with order > 0
// it will update event count accordingly
func (lv *LogView) mergeWrappedLines(event *logEventLine) *logEventLine {
	if event.order == 0 {
		return event
	}
	event = findFirstWrappedLine(event)
	event.order = 0
	event.start = 0
	event.lineCount = 1
	event.end = len(event.Runes)
	next := event.next
	if next == lv.lastEvent {
		lv.lastEvent = event
	}
	if next == lv.current {
		lv.current = event
	}
	if next == lv.top {
		lv.top = event
	}
	for next.next != nil && next.next.order > 1 { // find event with order <= 1, that will be
		next = next.next
		if next == lv.lastEvent {
			lv.lastEvent = event
		}
		if next == lv.current {
			lv.current = event
		}
		if next == lv.top {
			lv.top = event
		}
	}
	if next.next != nil {
		event.next = next.next
		next.next.previous = event
	} else {
		event.next = nil
	}
	return event
}

func (lv *LogView) insertAfter(node *logEventLine, new *logEventLine, adjustLineCount bool) *logEventLine {
	if node == nil {
		lv.firstEvent = new
		lv.lastEvent = new
		lv.top = new
		lv.current = new
	} else {
		new.previous = node
		new.next = node.next
		if node.next != nil {
			node.next.previous = new
		}
		node.next = new
		if lv.lastEvent == node {
			lv.lastEvent = new
		}
	}
	if adjustLineCount {
		lv.eventCount++
	}
	return new
}

func (lv *LogView) deleteEvent(event *logEventLine, adjustLineCount bool) {
	if event == nil {
		return
	}
	if event.next != nil {
		event.next.previous = event.previous
	}
	if event.previous != nil {
		event.previous.next = event.next
	}
	if event == lv.firstEvent {
		lv.firstEvent = event.next
	}
	if event == lv.lastEvent {
		lv.lastEvent = event.previous
	}
	if event == lv.top {
		if event.next == nil {
			lv.top = event.previous
		} else {
			lv.top = event.previous
		}
	}
	if event == lv.current {
		if event.next == nil {
			lv.current = event.previous
		} else {
			lv.current = event.previous
		}
	}
	if adjustLineCount {
		lv.eventCount--
	}
}

// replaceEvent chains all the events in the replacement slice and
// then replaces single event toReplace with new chain
func (lv *LogView) replaceEvent(toReplace *logEventLine, replacement []*logEventLine) *logEventLine {
	lastI := len(replacement) - 1
	for i, r := range replacement {
		if i > 0 {
			r.previous = replacement[i-1]
		}
		if i < lastI {
			r.next = replacement[i+1]
		}
	}
	if toReplace.previous != nil {
		toReplace.previous.next = replacement[0]
	}
	if toReplace.next != nil {
		toReplace.next.previous = replacement[lastI]
	}
	replacement[0].previous = toReplace.previous
	replacement[lastI].next = toReplace.next

	if toReplace == lv.firstEvent {
		lv.firstEvent = replacement[0]
	}
	if toReplace == lv.lastEvent {
		lv.lastEvent = replacement[lastI]
	}
	if toReplace == lv.current {
		lv.current = replacement[0]
	}
	if toReplace == lv.top {
		lv.top = replacement[0]
	}
	return replacement[lastI]
}

// unwrapLines removes all wrap lines
func (lv *LogView) unwrapLines() {
	event := lv.firstEvent
	for event != nil {
		isTop := event == lv.top
		event = lv.mergeWrappedLines(event)
		if isTop {
			lv.top = event
		}
		event = event.next
	}
}

// recolorizeLines calculates spans for strings
// This must be called after unwrapLines, otherwise it may panic on any wrapped line
func (lv *LogView) recolorizeLines() {
	event := lv.firstEvent
	for event != nil {
		lv.colorize(event)
		event = event.next
	}
}

// rewrapLines recalculates string wrapping
func (lv *LogView) rewrapLines() {
	event := lv.firstEvent
	for event != nil {
		isTop := event == lv.top
		event = lv.calculateWrap(event)
		if isTop {
			lv.top = event
		}
		event = event.next
	}
}

// drawEvent draws single event on a single line
func (lv *LogView) drawEvent(screen tcell.Screen, x int, y int, event *logEventLine) {
	if lv.showSource && lv.isHeaderPossible() {
		if event.order <= 1 {
			x = lv.printSource(screen, x, y, event) + 1
		} else {
			x += lv.sourceHeaderWidth()
		}
	}
	if lv.showTimestamp && lv.isHeaderPossible() {
		if event.order <= 1 {
			x = lv.printTimestamp(screen, x, y, event) + 1
		} else {
			x += lv.timestampHeaderWidth()
		}
	}

	if lv.highlightingEnabled {
		lv.printLogLine(screen, x, y, event)
	} else {
		lv.printLogLineNoHighlights(screen, x, y, event)
	}
}

func (lv *LogView) printSource(screen tcell.Screen, x int, y int, event *logEventLine) int {
	var source string
	if len(event.Source) > lv.sourceClipLength {
		source = event.Source[:lv.sourceClipLength]
	} else {
		source = fmt.Sprintf("%"+strconv.Itoa(lv.sourceClipLength)+"v", event.Source)
	}
	var style tcell.Style
	if lv.highlightCurrent && event == lv.current {
		style = lv.defaultStyle.Background(lv.currentBgColor)
	} else {
		style = lv.sourceStyle
	}

	lv.printSpecial(screen, x, y, event, source, style)

	return x + len(source) + 2
}

func (lv *LogView) printTimestamp(screen tcell.Screen, x int, y int, event *logEventLine) int {
	ts := event.Timestamp.Format(lv.timestampFormat)
	var style tcell.Style
	if lv.highlightCurrent && event == lv.current {
		style = lv.defaultStyle.Background(lv.currentBgColor)
	} else {
		style = lv.timestampStyle
	}
	return lv.printSpecial(screen, x, y, event, ts, style)
}

func (lv *LogView) printSpecial(screen tcell.Screen, x int, y int, event *logEventLine, ts string, style tcell.Style) int {
	printString(screen, x, y, ts, style)

	if lv.highlightCurrent && event == lv.current {
		style = lv.defaultStyle.Background(lv.currentBgColor)
	} else {
		style = lv.defaultStyle
	}
	printString(screen, x+len(ts)+1, y, "|", style)

	return x + len(ts) + 2
}

func (lv *LogView) printLogLine(screen tcell.Screen, x int, y int, event *logEventLine) {
	// find first styled span for the event
	spanIndex := 0
	for spanIndex < len(event.styleSpans) {
		if event.styleSpans[spanIndex].start <= event.start && event.styleSpans[spanIndex].end >= event.start {
			break
		}
		spanIndex++
	}
	if spanIndex == len(event.styleSpans) { // no colorization needed
		lv.printLogLineNoHighlights(screen, x, y, event)
		return
	}
	textPos := event.start
	i := x
	var style tcell.Style
	for textPos < event.end {
		style = event.styleSpans[spanIndex].style
		if lv.highlightCurrent && event == lv.current { // overwrite bg color for current selected event
			style = style.Background(lv.currentBgColor)
		}
		screen.SetCell(i, y, style, event.Runes[textPos])
		i++
		textPos++
		if textPos >= event.styleSpans[spanIndex].end {
			spanIndex++
		}
	}

	for i <= x+lv.pageWidth+5 {
		screen.SetCell(i, y, style, ' ')
		i++
	}

}

func (lv *LogView) printLogLineNoHighlights(screen tcell.Screen, x int, y int, event *logEventLine) {
	i := x
	style := lv.defaultStyle
	if lv.highlightCurrent && event == lv.current { // overwrite bg color for current selected event
		style = style.Background(lv.currentBgColor)
	}
	for pos := event.start; pos < event.end; pos++ {
		screen.SetCell(i, y, style, event.Runes[pos])
		i++
		if i >= lv.pageWidth {
			break
		}
	}
	for i <= x+lv.pageWidth {
		screen.SetCell(i, y, style, ' ')
		i++
	}
}

func (lv *LogView) clearLine(screen tcell.Screen, x, line int) {
	style := lv.defaultStyle
	i := x
	for i <= x+lv.pageWidth {
		screen.SetCell(i, line, style, ' ')
		i++
	}
}

func (lv *LogView) ensureEventLimit() {
	if lv.eventLimit == 0 {
		return
	}
	for lv.eventCount > lv.eventLimit {
		if lv.firstEvent != nil && lv.firstEvent.order > 0 {
			lv.mergeWrappedLines(lv.firstEvent)
		}
		lv.deleteEvent(lv.firstEvent, true)
	}
}

func (lv *LogView) scrollToStart() {
	lv.top = lv.firstEvent
	lv.current = lv.firstEvent
	lv.following = false
}

func (lv *LogView) scrollToEnd() {
	lv.top = lv.atOffset(lv.lastEvent, -(lv.pageHeight - 1))
	lv.current = lv.lastEvent
	lv.following = true
}

func (lv *LogView) scrollOneUp() {
	lv.following = false
	// if we're at the top of page or current highlighting is off then change the top
	if lv.current == lv.top || !lv.highlightCurrent {
		lv.top = lv.atOffset(lv.top, -1)
	}
	lv.current = lv.atOffset(lv.current, -1)
}

func (lv *LogView) scrollOneDown() {
	if lv.current == lv.lastEvent {
		lv.following = true
		return
	}
	lv.current = lv.atOffset(lv.current, 1)
	// if we're past end of page or current highlighting is off then change the top

	lv.following = false
}

func (lv *LogView) adjustTop() {
	if lv.distance(lv.current, lv.top) >= lv.pageHeight || !lv.highlightCurrent {
		lv.top = lv.atOffset(lv.top, 1)
	}
}

func (lv *LogView) scrollPageUp() {
	lv.top = lv.atOffset(lv.top, -lv.pageHeight)
	lv.current = lv.atOffset(lv.current, -lv.pageHeight)
	lv.following = false
}

func (lv *LogView) scrollPageDown() {
	lv.top = lv.atOffset(lv.top, lv.pageHeight)
	lv.current = lv.atOffset(lv.current, lv.pageHeight)
	if lv.current == lv.lastEvent {
		lv.following = true
		lv.top = lv.atOffset(lv.lastEvent, -(lv.pageHeight - 1))
	} else {
		lv.following = false
	}
}

func (lv *LogView) distance(start *logEventLine, target *logEventLine) int {
	limit := lv.pageHeight
	distance := 0
	event := start
	for limit > 0 {
		if event != target {
			event = event.previous
		} else {
			return distance
		}
		distance++
		limit--
	}
	return distance
}

func (lv *LogView) getBackgroundColor() tcell.Color {
	_, bg, _ := lv.defaultStyle.Decompose()
	return bg
}

func (lv *LogView) getTextColor() tcell.Color {
	fg, _, _ := lv.defaultStyle.Decompose()
	return fg
}

// headerWidth returns the width of the header of the log line
// If showSource or showTimestamp are enabled they create an additional header for the event
func (lv *LogView) headerWidth() int {
	w := 0
	if lv.showSource {
		w += lv.sourceHeaderWidth()
	}
	if lv.showTimestamp {
		w += lv.timestampHeaderWidth()
	}
	return w
}

// isHeaderPossible determines whether page has enough room to display a header
// if header is wider than half of the page then header would not be displayed
func (lv *LogView) isHeaderPossible() bool {
	return lv.headerWidth() < lv.fullPageWidth*1/2
}

func (lv *LogView) sourceHeaderWidth() int {
	return lv.sourceClipLength + 3
}

func (lv *LogView) timestampHeaderWidth() int {
	return len(lv.timestampFormat) + 3
}

func (lv *LogView) findByEventId(eventID string) *logEventLine {
	event := lv.firstEvent
	if eventID != "" {
		for event.EventID != eventID && event != nil {
			event = event.next
		}
	}
	return event
}
