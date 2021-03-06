package logview

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/gdamore/tcell/v2"
	"sort"
	"strings"
)

func (lv *LogView) defaultStyleEvent(event *logEventLine) *logEventLine {
	event.styleSpans = []styledSpan{
		{
			start: 0,
			end:   len(event.Runes),
			style: lv.defaultStyle,
		},
	}
	return event
}

type captureGroup struct {
	regexp2.Capture
	name string
}

type captureGroupSorter []captureGroup

func (cg captureGroupSorter) Len() int {
	return len(cg)
}

func (cg captureGroupSorter) Less(i, j int) bool {
	return cg[i].Index < cg[j].Index
}

func (cg captureGroupSorter) Swap(i, j int) {
	cg[i], cg[j] = cg[j], cg[i]
}

func (lv *LogView) colorize(event *logEventLine) *logEventLine {
	if event.order != 0 {
		panic(fmt.Errorf("cannot colorize wrapped line"))
	}
	defaultStyle := lv.defaultStyle
	useSpecialBg := false
	if lv.highlightLevels && event.Level != LogLevelInfo {
		useSpecialBg = true
		if event.Level == LogLevelWarning {
			defaultStyle = defaultStyle.Background(lv.warningBgColor)
		} else {
			defaultStyle = defaultStyle.Background(lv.errorBgColor)
		}
	}
	if lv.highlightingEnabled && lv.highlightPattern != nil {
		text := event.message()
		match, err := lv.highlightPattern.FindStringMatch(text)
		if err != nil || match == nil {
			return lv.defaultStyleEvent(event)
		}
		groups := make([]captureGroup, 0)
		for match != nil {
			for _, gr := range match.Groups() {
				if len(gr.Captures) > 0 {
					groups = append(groups, captureGroup{
						Capture: gr.Capture,
						name:    gr.Name,
					})
				}
			}
			match, err = lv.highlightPattern.FindNextMatch(match)
			if err != nil {
				return lv.defaultStyleEvent(event)
			}
		}
		// skip zero group as it represents the whole match  q
		groups = groups[1:]
		sort.Sort(captureGroupSorter(groups))
		event.styleSpans = lv.buildSpans([]rune(text), groups, defaultStyle, useSpecialBg)
	} else {
		event.styleSpans = []styledSpan{
			{
				start: 0,
				end:   len(event.Runes),
				style: defaultStyle,
			},
		}
	}
	return event
}

func (lv *LogView) groupNameToStyle(colorName string) tcell.Style {
	style := lv.defaultStyle
	colorPair := strings.Split(strings.ToLower(colorName), "_")
	if fg, ok := tcell.ColorNames[colorPair[0]]; ok {
		style = style.Foreground(fg)
	}
	if len(colorPair) > 1 {
		if bg, ok := tcell.ColorNames[colorPair[1]]; ok {
			style = style.Background(bg)
		}
	}
	return style
}

func (lv *LogView) buildSpans(text []rune, groups []captureGroup, defaultStyle tcell.Style, useDefaultBg bool) []styledSpan {
	currentPos := 0
	spans := make([]styledSpan, 0)

	_, dbg, _ := defaultStyle.Decompose()

	for _, group := range groups {
		if group.Index != currentPos {
			runeLen := len(text[currentPos:group.Index])
			beforeSpan := styledSpan{
				start: currentPos,
				end:   currentPos + runeLen,
				style: defaultStyle,
			}
			spans = append(spans, beforeSpan)
			currentPos = currentPos + runeLen
		}

		style := lv.groupNameToStyle(group.name)

		if useDefaultBg {
			style = style.Background(dbg)
		}

		matched := styledSpan{
			start: currentPos,
			end:   currentPos + group.Length,
			style: style,
		}
		spans = append(spans, matched)
		currentPos += group.Length
	}
	if currentPos < len(text)-1 {
		afterSpan := styledSpan{
			start: currentPos,
			end:   len(text),
			style: defaultStyle,
		}
		spans = append(spans, afterSpan)
	}
	return spans
}
