package main

import (
	"githib.com/uaraven/logview/logv"
	"github.com/gdamore/tcell/v2"
	"gitlab.com/tslocum/cview"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type UI struct {
	app     *cview.Application
	header  *cview.TextView
	logView *logv.LogView
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

	headerView := cview.NewTextView()

	logView := logv.NewLogView()
	logView.SetBorder(false)

	header := "[black]Log Group[-]:[navy]/ecs/legal-detection-staging[-]  [black]Time[-]:[navy]tailing[-]"
	headerView.SetBackgroundColor(tcell.ColorDarkGray)
	headerView.SetDynamicColors(true)
	headerView.SetBorder(false)
	headerView.SetText(header)

	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	flex.AddItem(headerView, 1, 1, false)
	flex.AddItem(logView, 0, 1, true)

	app.SetRoot(flex, true)

	return &UI{
		header:  headerView,
		logView: logView,
		app:     app,
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

	content, err := ioutil.ReadFile("test.log")

	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		event := &logv.LogEvent{
			EventID: strconv.Itoa(i),
			Level:   logv.LogLevelInfo,
			Message: line,
		}
		ui.logView.AppendLogEvent(event)
	}

	//event := logv.NewLogEvent("1", "20:17:51.894 [sqsTaskExecutor-10] ERROR  c.s.d.l.s.s.CopyrightDetectionService - This is the extra long line which originally said just this text that follows: Stored copyright data for pkg:npm/%40mpen/rollup-plugin-clean@0.1.8?checksum=sha1:097f0110bbc8aa5bc1026f2d689f45dcf98fcbc5&sonatype_repository=npmjs.org&type=tgz")
	//event.Level = logv.LogLevelWarning
	//ui.logView.AppendLogEvent(event)

	ui.Run()
}
