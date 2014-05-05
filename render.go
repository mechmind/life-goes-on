package main

import (
	"github.com/nsf/termbox-go"
)

const (
	TUI_DEFAULT_BG = termbox.ColorBlack
	TUI_DEFULT_FG  = termbox.ColorWhite

	TUI_SOLDIER_CHAR = '@'
	TUI_SOLDIER_FG   = termbox.ColorRed | termbox.AttrBold

	TUI_DAMSEL_CHAR = 'B'
	TUI_DAMSEL_FG   = termbox.ColorYellow

	TUI_ZED_CHAR = 'Z'
	TUI_ZED_FG   = termbox.ColorGreen

	TUI_FASTZED_CHAR = 'Z'
	TUI_FASTZED_FG   = termbox.ColorGreen | termbox.AttrBold

	TUI_CORPSE_BG = termbox.ColorRed
)

func pollEvents(events chan termbox.Event) {
	for {
		events <- termbox.PollEvent()
	}
}

func RunTUI(updates chan *Field) {
	var events = make(chan termbox.Event)
	go pollEvents(events)

	termbox.Init()
	defer termbox.Close()

	var currentPos CellCoord

	// recieve field view first
	var field = <-updates

	for {
		select {
		case newfield := <-updates:
			// send old field into field backbuffer
			select {
			case newfield.updates <- field:
			default:
			}

			field = newfield
			drawField(field, currentPos)
		case ev := <-events:
			switch ev.Type {
			case termbox.EventKey:
				switch {
				// move view left
				case ev.Ch == 'h':
					fallthrough
				case ev.Key == termbox.KeyArrowLeft:
					currentPos = currentPos.Add(-1, 0)

				// move view right
				case ev.Ch == 'l':
					fallthrough
				case ev.Key == termbox.KeyArrowRight:
					currentPos = currentPos.Add(1, 0)

				// move view up
				case ev.Ch == 'j':
					fallthrough
				case ev.Key == termbox.KeyArrowDown:
					currentPos = currentPos.Add(0, 1)

				// move view down
				case ev.Ch == 'k':
					fallthrough
				case ev.Key == termbox.KeyArrowUp:
					currentPos = currentPos.Add(0, -1)

				// quit
				case ev.Ch == 'q':
					return
				}
				drawField(field, currentPos)
			}
		}
	}
}

// render field chunk that we currently looking at
func drawField(f *Field, pos CellCoord) {
	sizex, sizey := termbox.Size()
	upperBound := pos.Add(sizex-1, sizey-1)

	termbox.Clear(TUI_DEFULT_FG, TUI_DEFAULT_BG)

	for _, up := range f.units {
		unitCell := up.coord.Cell()
		if !CheckCellCoordBounds(unitCell, pos, upperBound) {
			// unit is not visible
			continue
		}
		ch, fg, bg := getUnitView(up.unit)
		screenPos := unitCell.AddCoord(pos.Mul(-1))

		termbox.SetCell(screenPos.X, screenPos.Y, ch, fg, bg)
	}

	termbox.Flush()
}

func getUnitView(u Unit) (ch rune, fg, bg termbox.Attribute) {
	switch u.(type) {
	case *Soldier:
		return TUI_SOLDIER_CHAR, TUI_SOLDIER_FG, TUI_DEFAULT_BG
	case *Damsel:
		return TUI_DAMSEL_CHAR, TUI_DAMSEL_FG, TUI_DEFAULT_BG
	case *Zed:
		zed := u.(*Zed)
		if zed.nutrition > ZED_NUTRITION_FULL {
			return TUI_FASTZED_CHAR, TUI_FASTZED_FG, TUI_DEFAULT_BG
		} else {
			return TUI_ZED_CHAR, TUI_ZED_FG, TUI_DEFAULT_BG
		}
	case *Corpse:
		corpse := u.(*Corpse)
		ch, fg, _ := getUnitView(corpse.unit)
		return ch, fg, TUI_CORPSE_BG
	}
	return ' ', TUI_DEFULT_FG, TUI_DEFAULT_BG
}
