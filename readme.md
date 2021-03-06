# LogView widget for tview/cview

LogView widget for [tview](https://github.com/rivo/tview). 

It will also work with [cview](https://gitlab.com/tslocum/cview), just replace import `github.com/rivo/tview` with `gitlab.com/rivo/cview`. 

![](screen.png)

## Rationale

cview/tview TextView widget tries to recalculate highlighting and wrapping on every new line appended to the widget. Batch appends helps a little, but I was not able to achieve acceptable performance even when the line count in TextView's buffer was set to as little as 500.

LogView is designed for logs, so it supports very fast append operation and takes special care to calculate highlighting and line wrapping only once for each log event added.

LogView operates on LogEvent structures, not on text lines, this allows keeping track of which line belongs to which event even with wrapping enabled and easy navigation to specific log event by its ID or timestamp.

## Capabilities

LogView supports:

- [x] tailing logs
- [x] limiting the number of log events stored in log view
- [x] highlighting error/warning events (with customizable colors)
- [x] custom highlighting of parts of log messages
- [x] scrolling to event id
- [x] scrolling to timestamp
- [x] optional display of log event source and timestamp separately from main message
- [x] keyboard and mouse scrolling
- [x] selection of log event with a keyboard or mouse with a callback on selection change 
- [x] merging of continuation events (i.e. multiline java stack-traces can be treated as one log event)
- [x] velocity graph

## Performance notes

LogView attempts to minimize the number of calculations performed. For each log event, the line wrapping and colour
highlighting are calculated only once, at the moment the event is appended to the log view. This allows for very
fast appends, but also means the whole log view can become stale if widget size or colour settings change.

Widget size changes are handled automatically. If line wrapping is disabled, then no additional work has to be done, otherwise
line wrapping are recalculated as needed.

Recalculating highlights changes is a more expensive operation, so it is not handled automatically. To force recalculation
of highlights for all the log events call `LogView.RefreshHighlighs()` method.

Changes to any of the highlights or default Log view style would require recalculation. Changes to the background colour of
current event or error and warning level events do not require recalculation.

## Event Message Highlighting

LogView doesn't use tview color tags, mostly because they are an unnecessary step in colorizing event message. LogView
accepts regular expression to extract parts of the log message and apply styles to them.

Named capture groups define parts of the message that should be highlighted, and the name of the group defines the colors.

Group name must match `foreground_background` pattern. Background color is optional and colors must match names of "extended" 
web colors. Defining colors in hex is not yet supported.

**Note**: Regular expression syntax for color highlighting is different from standard Go regexp syntax, you can use
lookahead and lookbehind but setting regexp flags with (?iUa), etc is not supported. Using lookahead/behind can be a 
significant performance hit though. A general rule is to try to stick to the simplest expressions possible. 
Use non-capturing groups for everything, except for the things you need highlighted.
                                                                       
See [regexp2 readme](https://github.com/dlclark/regexp2) for more details.  

Examples (note the use of non-capturing groups):
    
    (?P<lavender>\d{2}(?:[:.-]\d{2,3})+) - match time tokens like 2021-03-05 or 12:23:44.332 and highlight them with 
                                           lavender color
    (?:\b(?P<white_lightsalmon>info|warning|error|trace|debug)\b) - match debug level as a separate word and highlight
                                           it as white on reddish color

## LogVelocityView Widget

Log velocity widget displays bar chart of number of log events per time period. Widget can show count for all events or
only events of the certain level.

Note. Many fonts will have weird line gaps in the block characters. Hack is one of the best in this regard.
