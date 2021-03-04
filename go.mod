module github.com/uaraven/logview

go 1.16

require (
	github.com/araddon/dateparse v0.0.0-20210207001429-0eec95c9db7e
	github.com/gdamore/tcell/v2 v2.2.0
	github.com/uaraven/tview v0.0.0
	gitlab.com/tslocum/cbind v0.1.4
)

replace (
	github.com/uaraven/tview => ../tview
)