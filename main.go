package main

import (
	"fmt"
	"githib.com/uaraven/logview/logv"
	"github.com/gdamore/tcell/v2"
	"gitlab.com/tslocum/cview"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type UI struct {
	app       *cview.Application
	histogram *logv.LogVelocityView
	logView   *logv.LogView
}

// CreateAppUI creates base UI layout
func CreateAppUI() *UI {
	app := cview.NewApplication()
	app.EnableMouse(true)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if cview.HitShortcut(event, []string{"q"}) {
			app.Stop()
		}
		return event
	})

	logView := logv.NewLogView()
	logView.SetBorder(false)

	histogramView := logv.NewLogVelocityView(10 * time.Second)

	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	flex.AddItem(logView, 0, 1, true)
	flex.AddItem(histogramView, 5, 1, false)

	app.SetRoot(flex, true)

	return &UI{
		histogram: histogramView,
		logView:   logView,
		app:       app,
	}
}

// Run starts the application event loop
func (ui *UI) Run() {
	if err := ui.app.Run(); err != nil {
		log.Fatalln(err)
	}
}

// Stop the UI
func (ui *UI) Stop() {
	ui.app.Stop()
}

func main() {
	ui := CreateAppUI()

	ui.logView.SetHighlightPattern(`(?P<ts>\d{2}:\d{2}:\d{2}.\d{3})\s+\[(?P<thread>.*)\]\s+(?P<level>\S+)\s+(?P<class>[a-zA-Z0-9_.]+).*(?:in (?P<elapsed>\d+)ms)?`)
	ui.logView.SetHighlightColorFg("ts", tcell.ColorDarkCyan)
	ui.logView.SetHighlightColorFg("thread", tcell.ColorDarkGreen)
	ui.logView.SetHighlightColorFg("level", tcell.ColorYellow)
	ui.logView.SetHighlightColorFg("class", tcell.ColorCadetBlue)
	ui.logView.SetHighlightColor("elapsed", tcell.ColorGreenYellow, tcell.ColorDarkKhaki)
	ui.logView.SetHighlighting(true)
	ui.logView.SetLevelHighlighting(true)
	ui.logView.SetHighlightCurrentEvent(true)
	ui.logView.SetShowTimestamp(true)
	ui.logView.SetShowSource(true)

	content, err := ioutil.ReadFile("test.log")

	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(content), "\n")

	start := time.Now().Add(-15 * time.Minute)
	current := start

	for i, line := range lines {
		event := &logv.LogEvent{
			EventID:   strconv.Itoa(i),
			Level:     logv.LogLevelInfo,
			Message:   line,
			Source:    "S " + strconv.Itoa(i),
			Timestamp: current,
		}
		current = current.Add(time.Duration(rand.Float64() / 2 * float64(time.Second)))
		ui.logView.AppendLogEvent(event)
		ui.histogram.AppendLogEvent(event)
	}

	diff := current.Sub(start)

	//event := logv.NewLogEvent("1", "20:17:51.894 [sqsTaskExecutor-10] ERROR  c.s.d.l.s.s.CopyrightDetectionService - This is the extra long line which originally said just this text that follows: Stored copyright data for pkg:npm/%40mpen/rollup-plugin-clean@0.1.8?checksum=sha1:097f0110bbc8aa5bc1026f2d689f45dcf98fcbc5&sonatype_repository=npmjs.org&type=tgz")
	//event.Level = logv.LogLevelWarning
	//ui.logView.AppendLogEvent(event)

	ui.Run()

	fmt.Println(diff)
}
