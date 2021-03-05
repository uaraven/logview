package logview

import (
	"github.com/gdamore/tcell/v2"
	gui "github.com/rivo/tview"
	"strconv"
	"testing"
)

const eventCount = 50

func BenchmarkTextViewReindexOn(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	tv := gui.NewTextView()
	tv.SetDynamicColors(true)

	events := randomBenchStrings(eventCount)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, s := range events {
			_, err := tv.Write([]byte(s))
			if err != nil {
				panic(err)
			}
			tv.Draw(screen)
		}
	}
}

func BenchmarkTextViewReindexOff(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	tv := gui.NewTextView()
	//tv.SetReindexBuffer(false)
	tv.SetDynamicColors(true)

	events := randomBenchStrings(eventCount)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, s := range events {
			_, err := tv.Write([]byte(s))
			if err != nil {
				panic(err)
			}
			tv.Draw(screen)
		}
	}
}

func BenchmarkTextViewSingleDraw(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	tv := gui.NewTextView()
	//tv.SetReindexBuffer(false)
	tv.SetDynamicColors(true)

	events := randomBenchStrings(eventCount)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, s := range events {
			_, err := tv.Write([]byte(s))
			if err != nil {
				panic(err)
			}
		}
	}
	tv.Draw(screen)
}

func randomBenchStrings(count int) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)
		result[i] = "[red]e" + idx + "[-][white]Event #" + idx + "[-][blue+h] ![-]"
	}
	return result
}
