package main

import (
	"github.com/nsf/termbox-go"
)

const (
	TUI_DEFAULT_BG = termbox.ColorBlack
	TUI_DEFAULT_FG = termbox.ColorWhite

	TUI_SOLDIER_CHAR = '@'
	TUI_SOLDIER_FG   = termbox.ColorRed | termbox.AttrBold

	TUI_DAMSEL_CHAR = 'B'
	TUI_DAMSEL_FG   = termbox.ColorYellow

	TUI_FASTDAMSEL_CHAR = 'B'
	TUI_FASTDAMSEL_FG   = termbox.ColorYellow | termbox.AttrBold

	TUI_ZED_CHAR = 'Z'
	TUI_ZED_FG   = termbox.ColorGreen

	TUI_FASTZED_CHAR = 'Z'
	TUI_FASTZED_FG   = termbox.ColorGreen | termbox.AttrBold

	TUI_CORPSE_BG = termbox.ColorRed

	TUI_WALL_CHAR      = '#'
	TUI_FLAT_CHAR      = ' '
	TUI_OFFSCREEN_CHAR = TUI_WALL_CHAR

	TUI_POS_STEP = 5

	// FIXME(pathfind)
	TUI_PATHFIND_OPEN_BG   = termbox.ColorCyan
	TUI_PATHFIND_CLOSED_BG = termbox.ColorYellow
	TUI_PATHFIND_PATH_BG   = termbox.ColorRed
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
					currentPos = currentPos.Add(-TUI_POS_STEP, 0)

				// move view right
				case ev.Ch == 'l':
					fallthrough
				case ev.Key == termbox.KeyArrowRight:
					currentPos = currentPos.Add(TUI_POS_STEP, 0)

				// move view up
				case ev.Ch == 'j':
					fallthrough
				case ev.Key == termbox.KeyArrowDown:
					currentPos = currentPos.Add(0, TUI_POS_STEP)

				// move view down
				case ev.Ch == 'k':
					fallthrough
				case ev.Key == termbox.KeyArrowUp:
					currentPos = currentPos.Add(0, -TUI_POS_STEP)

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

	termbox.Clear(TUI_DEFAULT_FG, TUI_DEFAULT_BG)

	var fieldZero = CellCoord{0, 0}
	var fieldMax = CellCoord{f.xSize - 1, f.ySize - 1}
	// render walls
	for i := pos.X; i < upperBound.X; i++ {
		for j := pos.Y; j < upperBound.Y; j++ {
			tileCell := CellCoord{i, j}
			screenPos := tileCell.AddCoord(pos.Mul(-1))
			if !CheckCellCoordBounds(tileCell, fieldZero, fieldMax) {
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_OFFSCREEN_CHAR,
					TUI_DEFAULT_FG, TUI_DEFAULT_BG)
			} else {
				if f.CellAt(tileCell).passable {
					termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
						TUI_DEFAULT_FG, TUI_DEFAULT_BG)
				} else {
					termbox.SetCell(screenPos.X, screenPos.Y, TUI_WALL_CHAR,
						TUI_DEFAULT_FG, TUI_DEFAULT_BG)
				}
			}
		}
	}

	// render units
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

	// render pathfind
	// FIXME(pathfind)
	if p := f.pathfinder; p != nil {
		for coord, cell := range p.cells {
			if !CheckCellCoordBounds(coord, pos, upperBound) {
				continue
			}
			screenPos := coord.AddCoord(pos.Mul(-1))
			switch {
			case cell.closed:
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
					TUI_DEFAULT_FG, TUI_PATHFIND_CLOSED_BG)
			case cell.open:
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
					TUI_DEFAULT_FG, TUI_PATHFIND_OPEN_BG)
			}
		}
		for _, coord := range p.path {
			if !CheckCellCoordBounds(coord, pos, upperBound) {
				continue
			}
			screenPos := coord.AddCoord(pos.Mul(-1))
			termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
				TUI_DEFAULT_FG, TUI_PATHFIND_PATH_BG)
		}
	}

	termbox.Flush()
}

func getUnitView(u Unit) (ch rune, fg, bg termbox.Attribute) {
	switch u.(type) {
	case *Soldier:
		return TUI_SOLDIER_CHAR, TUI_SOLDIER_FG, TUI_DEFAULT_BG
	case *Damsel:
		dam := u.(*Damsel)
		if dam.adrenaline > 0 {
			return TUI_FASTDAMSEL_CHAR, TUI_FASTDAMSEL_FG, TUI_DEFAULT_BG
		} else {
			return TUI_DAMSEL_CHAR, TUI_DAMSEL_FG, TUI_DEFAULT_BG
		}
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
	return ' ', TUI_DEFAULT_FG, TUI_DEFAULT_BG
}
