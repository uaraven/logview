package logv

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"gitlab.com/tslocum/cview"
	"sync"
	"time"
)

// LogVelocityView is a bar chart to display number of log events per time period
type LogVelocityView struct {
	*cview.Box

	defaultStyle tcell.Style
	errorColor   tcell.Color
	warningColor tcell.Color

	showLogLevel LogLevel
	bucketWidth  int64
	infoBuckets  map[int64]int
	errorBuckets map[int64]int
	warnBuckets  map[int64]int
	height       int
	width        int

	anchor *int64

	sync.RWMutex
}

const valuesPerBlock = 8
const minWidthToDisplayYAxis = 20

var blocks = []rune{
	'\u2581', // U+2581 1/8
	'\u2582', // U+2582 2/8
	'\u2583', // U+2583 3/8
	'\u2584', // U+2584 4/8
	'\u2585', // U+2585 5/8
	'\u2586', // U+2586 6/8
	'\u2587', // U+2587 7/8
	'\u2588', // U+2588 8/8
}

// NewLogVelocityView creates a new log velocity view with a defined bucket time frame
func NewLogVelocityView(bucketWidth time.Duration) *LogVelocityView {
	return &LogVelocityView{
		Box:          cview.NewBox(),
		bucketWidth:  int64(bucketWidth.Seconds()),
		infoBuckets:  make(map[int64]int),
		warnBuckets:  make(map[int64]int),
		errorBuckets: make(map[int64]int),
		defaultStyle: tcell.StyleDefault.Foreground(cview.Styles.PrimaryTextColor).Background(tcell.Color239),
		errorColor:   tcell.ColorIndianRed,
		warningColor: tcell.ColorSaddleBrown,
		showLogLevel: LogLevelAll,
		anchor:       nil,
	}
}

// AppendLogEvents adds event to a velocity chart
func (lh *LogVelocityView) AppendLogEvent(event *LogEvent) {
	lh.Lock()
	defer lh.Unlock()

	key := event.Timestamp.Unix() / lh.bucketWidth

	var b map[int64]int
	switch event.Level {
	case LogLevelError:
		b = lh.errorBuckets
	case LogLevelWarning:
		b = lh.warnBuckets
	default:
		b = lh.infoBuckets
	}

	if v, ok := b[key]; ok {
		b[key] = v + 1
	} else {
		b[key] = 1
	}
}

// SetShowLogLevel sets the log level of events that should be displayed in the velocity view
//
// Supported values are:
//
//  - LogLevelInfo - show all events but warning and errors
//
//	- LogLevelWarning
//
//  - LogLevelError
//
//  - LogLevelAll - show all events
func (lh *LogVelocityView) SetShowLogLevel(logLevel LogLevel) {
	lh.Lock()
	defer lh.Unlock()

	lh.showLogLevel = logLevel
}

// GetShowLogLevel returns the log level of events that are be displayed in the velocity view
func (lh *LogVelocityView) GetShowLogLevel() LogLevel {
	lh.RLock()
	defer lh.RUnlock()

	return lh.showLogLevel
}

// Draw draws this primitive onto the screen.
func (lh *LogVelocityView) Draw(screen tcell.Screen) {
	if !lh.GetVisible() {
		return
	}

	lh.Box.Draw(screen)

	lh.Lock()
	defer lh.Unlock()

	// Get the available size.
	x, y, width, height := lh.GetInnerRect()

	if height == 0 {
		return
	}

	lh.width = width
	lh.height = height

	var values []int

	key := lh.timeAnchor()
	if width < minWidthToDisplayYAxis {
		values = lh.values(key, width)
	} else {
		values = lh.values(key, width-6)
	}
	maxV := lh.max(values)

	if width > 20 {
		x, width = lh.drawValueAxis(screen, x, y, maxV)
	}
	if height > 1 {
		lh.drawTimeAxis(screen, x, y, width, height, key)
		height--
	}
	lh.drawHistogram(screen, x, y, width, height, values, maxV)
}

// ****************
// Internal methods

func (lh *LogVelocityView) bucketValue(bucket map[int64]int, key int64) int {
	if v, ok := bucket[key]; ok {
		return v
	} else {
		return 0
	}
}

func (lh *LogVelocityView) values(key int64, count int) []int {
	results := make([]int, count)
	for i := count - 1; i >= 0; i-- {
		var value int
		switch lh.showLogLevel {
		case LogLevelWarning:
			value = lh.bucketValue(lh.warnBuckets, key)
		case LogLevelError:
			value = lh.bucketValue(lh.errorBuckets, key)
		case LogLevelInfo:
			value = lh.bucketValue(lh.infoBuckets, key)
		case LogLevelAll:
			value = lh.bucketValue(lh.infoBuckets, key) + lh.bucketValue(lh.warnBuckets, key) +
				lh.bucketValue(lh.errorBuckets, key)
		default:
			value = 0
		}
		results[i] = value
		key = key - 1
	}
	return results
}

func (lh *LogVelocityView) drawHistogram(screen tcell.Screen, x, y, width, height int, values []int, maxV int) {
	var scale float64
	if maxV == 0 {
		scale = float64(maxV)
	} else {
		scale = float64(height*valuesPerBlock) / float64(maxV)
	}
	// normalize values to available height
	for i, v := range values {
		values[i] = int(float64(v) * scale)
	}
	style := lh.defaultStyle
	switch lh.showLogLevel {
	case LogLevelWarning:
		style = style.Foreground(lh.warningColor)
	case LogLevelError:
		style = style.Foreground(lh.errorColor)
	}

	index := len(values) - 1
	i := x + width - 1
	for i >= 0 && index >= 0 {
		v := values[index]
		for j := y + height - 1; j >= y; j-- {
			if v > valuesPerBlock {
				screen.SetCell(i, j, style, blocks[7])
			} else {
				if v > 0 {
					block := v - 1
					screen.SetCell(i, j, style, blocks[block])
				} else {
					screen.SetCell(i, j, style, ' ')
				}
			}
			v -= valuesPerBlock
		}
		i--
		index--
	}
}

// drawTimeAxis draws X-axis with duration marks
func (lh *LogVelocityView) drawTimeAxis(screen tcell.Screen, x int, y int, width int, height int, key int64) {
	tickDuration := lh.bucketWidth * 20
	current := key
	for i := 0; i < width; i++ {
		screen.SetCell(x+i, y+height-1, lh.defaultStyle, ' ')
	}
	i := width - 1
	yp := y + height - 1
	for i >= 0 {
		dur := "-" + durationToString(key-current)
		i -= len(dur)
		if i <= 0 {
			break
		}
		printString(screen, x+i, yp, dur, lh.defaultStyle)
		screen.SetCell(x+i+len(dur), yp, lh.defaultStyle, '⭡')
		current -= tickDuration
		i -= 20 - (len(dur))
	}
}

// drawValueAxis draws Y-axis with marks for zero and max event count per bucket
// it returns new minimal x coordinate. Y-axis takes 6 characters out of screen real estate
func (lh *LogVelocityView) drawValueAxis(screen tcell.Screen, x int, y int, maxValue int) (int, int) {
	valueS := formatValue(maxValue)
	printString(screen, x, y, valueS, lh.defaultStyle)
	printString(screen, x+4, y, " ┌", lh.defaultStyle)
	for j := y + 1; j < y+lh.height-1; j++ {
		printString(screen, x, j, "     │", lh.defaultStyle)
	}
	printString(screen, x, y+lh.height-1, "   0 └", lh.defaultStyle)
	return x + 6, lh.width - 6
}

func (lh *LogVelocityView) max(values []int) int {
	m := 0
	for _, val := range values {
		v := val
		if v > m {
			m = v
		}
	}
	return m
}

func (lh *LogVelocityView) timeAnchor() int64 {
	if lh.anchor == nil {
		return time.Now().Unix() / lh.bucketWidth
	} else {
		return *lh.anchor / lh.bucketWidth
	}
}

func durationToString(d int64) string {
	minutes := d / 60
	seconds := d % 60

	hours := minutes / 60
	minutes = minutes % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
