# LogView widget for cview

LogView widget for [cview](https://gitlab.com/tslocum/cview). Might also work with [tview](https://github.com/rivo/tview) 
although that was not tested.

## Rationale

cview/tview TextView widget tries to recalculate highlighting and wrapping on every new line appended to the
widget. Batch appends helps a little, but I was not able to achieve acceptable performance even with number of
lines as small as 500.

LogView designed for append-only logs and takes special care to calculate highlighting only once for each
log event added. It also uses different internal representation for data that allows very quick 
"append last"/"delete first" operations.

LogView operates on LogEvent structures, not on text lines, this allows keeping track of which line belongs to
which event even with wrapping enabled and easy navigation to specific log event by its ID or timestamp.

## Capabilities

LogView supports:

 - [x] tailing logs
 - [x] limiting the number of log events stored in logview
 - [x] highlighting error/warning events (with customizable colors)
 - [x] custom highlighting of parts of log messages
 - [x] scrolling to event id
 - [x] scrolling to timestamp  
 - [ ] optional display of log event source and timestamp separately from main message
 - [x] keyboard and mouse scrolling (mouse not working for unknown reason)
 - [x] selection of log event
 - [ ] velocity histogram
 