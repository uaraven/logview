package logv

import (
	"github.com/gdamore/tcell/v2"
	"gitlab.com/tslocum/cview"
	"strconv"
	"testing"
)

func BenchmarkTextView(b *testing.B) {
	screen := tcell.NewSimulationScreen("UTF-8")
	lv := cview.NewTextView()

	for n := 0; n < b.N; n++ {
		for _, s := range randomBenchStrings(40) {
			_, err := lv.Write([]byte(s))
			if err != nil {
				panic(err)
			}
			lv.Draw(screen)
		}
	}
}

func randomBenchStrings(count int) []string {
	result := make([]string, count)
	for i := 0; i < count; i++ {
		idx := strconv.Itoa(i)
		result[i] = "[red]e" + idx + "[-][white]Event #" + idx + "[-][blue+h] ![-]"
	}
	return result
}
