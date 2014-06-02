package main

import (
	"fmt"
	"github.com/nsf/termbox-go"
	"strings"
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

	// squad HUD
	TUI_MOVE_TARGET_CHAR = '+'
	TUI_MOVE_TARGET_FG   = termbox.ColorGreen | termbox.AttrBold

	TUI_GREN_TARGET_CHAR = '*'
	TUI_GREN_TARGET_FG   = termbox.ColorRed | termbox.AttrBold

	TUI_FLYING_GREN_TARGET_CHAR = '*'
	TUI_FLYING_GREN_TARGET_FG   = termbox.ColorYellow

	// FIXME(pathfind)
	TUI_PATHFIND_OPEN_BG   = termbox.ColorCyan
	TUI_PATHFIND_CLOSED_BG = termbox.ColorYellow
	TUI_PATHFIND_PATH_BG   = termbox.ColorRed

	TUI_CURSOR_MARGIN = 5
	
	// status
	TUI_STATUS_FIRE_FG = termbox.ColorRed
)

var boomingColors = [SOL_GREN_TICK_CAP + 1]struct {
	fg, bg termbox.Attribute
	ch     rune
}{
	{},
	{termbox.ColorWhite, termbox.ColorWhite, ' '},
	{termbox.ColorYellow, termbox.ColorYellow, ' '},
	{termbox.ColorRed, termbox.ColorDefault, '*'},
}

func pollEvents(events chan termbox.Event) {
	for {
		events <- termbox.PollEvent()
	}
}

func tb2cell() CellCoord {
	x, y := termbox.Size()
	return CellCoord{x, y}
}

func handleCursorMove(size, pos, cursor CellCoord) CellCoord {
	low := pos.Add(TUI_CURSOR_MARGIN, TUI_CURSOR_MARGIN)
	high := pos.AddCoord(size).Add(-TUI_CURSOR_MARGIN, -TUI_CURSOR_MARGIN)
	if !CheckCellCoordBounds(cursor, low, high) {
		// cursor is too close to border, move window
		switch {
		case cursor.X < low.X:
			pos.X -= TUI_POS_STEP
		case cursor.X > high.X:
			pos.X += TUI_POS_STEP
		case cursor.Y < low.Y:
			pos.Y -= TUI_POS_STEP
		case cursor.Y > high.Y:
			pos.Y += TUI_POS_STEP
		}
	}
	return pos
}

func sendOrder(orders chan Order, o Order) {
	select {
	case orders <- o:
	default:
	}
}


type squadView struct {
	fireState int
	movingTo  CellCoord
	grenTo    CellCoord
	automove  bool
}

func RunTUI(updates chan *Field, gameStateChan chan int, orders chan Order) {
	var events = make(chan termbox.Event)
	go pollEvents(events)

	prepareTerminal()

	termbox.Init()
	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	defer termbox.Close()

	var currentPos CellCoord
	var size = tb2cell()
	var cursorPos = CellCoord{size.X / 2, size.Y / 2} // center cursor
	//termbox.SetCursor(cursorPos.X, cursorPos.Y)

	// FIXME: hardcoded squad values
	var sv = squadView{fireState: ORDER_FIRE}

	// recieve field view first
	var field = <-updates

	var gameState = GAME_WAIT

	for {
		select {
		case gameState = <-gameStateChan:
		case newfield := <-updates:
			// send old field into field backbuffer
			select {
			case newfield.updates <- field:
			default:
			}

			field = newfield
			// update rendering state
			// handle grens
			for _, gren := range field.grens {
				if gren.from.Cell() == sv.grenTo {
					sv.grenTo = CellCoord{0, 0}
					break
				}
			}
			drawField(field, currentPos, sv, gameState)
		case ev := <-events:
			switch ev.Type {
			case termbox.EventMouse:
				cursorPos = currentPos.Add(ev.MouseX, ev.MouseY)
				//termbox.SetCursor(ev.MouseX, ev.MouseY)
				switch {
				case ev.Key == termbox.MouseLeft:
					if (CheckCellCoordBounds(cursorPos, CellCoord{0, 0}, CellCoord{1024, 1024}) &&
						field.CellAt(cursorPos).passable) {
						sendOrder(orders, Order{ORDER_MOVE, cursorPos})
						sv.movingTo = cursorPos
					}
				case ev.Key == termbox.MouseRight:
					sendOrder(orders, Order{ORDER_GREN, cursorPos})
					sv.grenTo = cursorPos
				}

			case termbox.EventKey:
				switch {
				// move view left
				case ev.Ch == 'h':
					cursorPos = cursorPos.Add(-1, 0)

				// move view right
				case ev.Ch == 'l':
					cursorPos = cursorPos.Add(1, 0)

				// move view up
				case ev.Ch == 'j':
					cursorPos = cursorPos.Add(0, 1)

				// move view down
				case ev.Ch == 'k':
					cursorPos = cursorPos.Add(0, -1)

				// direct moving window
				case ev.Key == termbox.KeyArrowLeft:
					fallthrough
				case ev.Ch == 'a':
					cursorPos = cursorPos.Add(-TUI_POS_STEP, 0)
					currentPos = currentPos.Add(-TUI_POS_STEP, 0)

				case ev.Key == termbox.KeyArrowRight:
					fallthrough
				case ev.Ch == 'd':
					cursorPos = cursorPos.Add(TUI_POS_STEP, 0)
					currentPos = currentPos.Add(TUI_POS_STEP, 0)

				case ev.Key == termbox.KeyArrowDown:
					fallthrough
				case ev.Ch == 's':
					cursorPos = cursorPos.Add(0, TUI_POS_STEP)
					currentPos = currentPos.Add(0, TUI_POS_STEP)

				case ev.Key == termbox.KeyArrowUp:
					fallthrough
				case ev.Ch == 'w':
					cursorPos = cursorPos.Add(0, -TUI_POS_STEP)
					currentPos = currentPos.Add(0, -TUI_POS_STEP)

				// big leaps
				// move view left
				case ev.Ch == 'H':
					cursorPos = cursorPos.Add(-TUI_POS_STEP, 0)

				// move view right
				case ev.Ch == 'L':
					cursorPos = cursorPos.Add(TUI_POS_STEP, 0)

				// move view up
				case ev.Ch == 'J':
					cursorPos = cursorPos.Add(0, TUI_POS_STEP)

				// move view down
				case ev.Ch == 'K':
					cursorPos = cursorPos.Add(0, -TUI_POS_STEP)

				// orders
				case ev.Key == termbox.KeySpace:
					sendOrder(orders, Order{ORDER_MOVE, cursorPos})
					sv.movingTo = cursorPos
					sv.automove = false

				case ev.Ch == 'g':
					fallthrough
				case ev.Ch == 'G':
					sendOrder(orders, Order{ORDER_GREN, cursorPos})
					sv.grenTo = cursorPos

				case ev.Ch == 'f':
					fallthrough
				case ev.Ch == 'F':
					sv.fireState = toggleFireState(sv.fireState)
					sendOrder(orders, Order{sv.fireState, cursorPos})

				case ev.Ch == 'p':
					fallthrough
				case ev.Ch == 'P':
					sendOrder(orders, Order{ORDER_AUTOMOVE, cursorPos})
					sv.automove = true

				// quit
				case ev.Key == termbox.KeyF10:
					return
				}
				currentPos = handleCursorMove(size, currentPos, cursorPos)
				//relativeCursorPos := cursorPos.AddCoord(currentPos.Mult(-1))
				//termbox.SetCursor(relativeCursorPos.X, relativeCursorPos.Y)
				drawField(field, currentPos, sv, gameState)
			case termbox.EventResize:
				size = tb2cell()
				currentPos = handleCursorMove(size, currentPos, cursorPos)
				//relativeCursorPos := cursorPos.AddCoord(currentPos.Mult(-1))
				//termbox.SetCursor(relativeCursorPos.X, relativeCursorPos.Y)
				drawField(field, currentPos, sv, gameState)
			}
		}
	}
}

// render field chunk that we currently looking at
func drawField(f *Field, pos CellCoord, sv squadView, gameState int) {
	// 2 lines are reserved for messages and status bars
	upperBound := tb2cell().Add(-1, -1).AddCoord(pos)

	termbox.Clear(TUI_DEFAULT_FG, TUI_DEFAULT_BG)

	var fieldZero = CellCoord{0, 0}
	var fieldMax = CellCoord{f.xSize - 1, f.ySize - 1}
	// render walls
	for i := pos.X; i < upperBound.X; i++ {
		for j := pos.Y; j < upperBound.Y; j++ {
			tileCell := CellCoord{i, j}
			screenPos := tileCell.AddCoord(pos.Mult(-1))
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
		screenPos := unitCell.AddCoord(pos.Mult(-1))

		termbox.SetCell(screenPos.X, screenPos.Y, ch, fg, bg)
	}

	// render squad state
	if (sv.movingTo != CellCoord{0, 0}) && CheckCellCoordBounds(sv.movingTo, pos, upperBound) {
		screenPos := sv.movingTo.AddCoord(pos.Mult(-1))
		termbox.SetCell(screenPos.X, screenPos.Y, TUI_MOVE_TARGET_CHAR,
			TUI_MOVE_TARGET_FG, TUI_DEFAULT_BG)
	}

	// render grens
	for _, gren := range f.grens {
		if gren.booming == 0 {
			// flying gren
			if CheckCellCoordBounds(gren.from.Cell(), pos, upperBound) {
				screenPos := gren.from.Cell().AddCoord(pos.Mult(-1))
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLYING_GREN_TARGET_CHAR,
					TUI_FLYING_GREN_TARGET_FG, TUI_DEFAULT_BG)
			}
		} else {
			// explosion
			center := gren.to
			for i := -SOL_GREN_RADIUS; i <= SOL_GREN_RADIUS; i++ {
				for j := -SOL_GREN_RADIUS; j <= SOL_GREN_RADIUS; j++ {
					cellCoord := center.Cell().Add(i, j)
					screenPos := cellCoord.AddCoord(pos.Mult(-1))
					if CheckCellCoordBounds(cellCoord, pos, upperBound) &&
						center.Distance(cellCoord.UnitCenter()) < SOL_GREN_RADIUS &&
						f.HaveLOS(center, cellCoord.UnitCenter()) {
						// in a range and visible
						boomingView := boomingColors[gren.booming]
						termbox.SetCell(screenPos.X, screenPos.Y,
							boomingView.ch, boomingView.fg, boomingView.bg)
					}
				}
			}
		}
	}

	if (sv.grenTo != CellCoord{0, 0}) && CheckCellCoordBounds(sv.grenTo, pos, upperBound) {
		screenPos := sv.grenTo.AddCoord(pos.Mult(-1))
		termbox.SetCell(screenPos.X, screenPos.Y, TUI_GREN_TARGET_CHAR,
			TUI_GREN_TARGET_FG, TUI_DEFAULT_BG)
	}
	// render pathfind
	// FIXME(pathfind)
	if p := f.pathfinder; p != nil {
		for coord, cell := range p.cells {
			break
			if !CheckCellCoordBounds(coord, pos, upperBound) {
				continue
			}
			screenPos := coord.AddCoord(pos.Mult(-1))
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
			break
			if !CheckCellCoordBounds(coord, pos, upperBound) {
				continue
			}
			screenPos := coord.AddCoord(pos.Mult(-1))
			termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
				TUI_DEFAULT_FG, TUI_PATHFIND_PATH_BG)
		}
	}

	// render status and message bars
	var statusPos int
	var fireState = "[ %s ]"
	switch sv.fireState {
	case ORDER_FIRE:
		fireState = fmt.Sprintf(fireState, "STAY_FIRE")
	case ORDER_SEMIFIRE:
		fireState = fmt.Sprintf(fireState, "RUN_FIRE")
	case ORDER_NOFIRE:
		fireState = fmt.Sprintf(fireState, "NO_FIRE")
	}

	yPos := tb2cell().Y-1
	statusPos = writeTermString(fireState, TUI_STATUS_FIRE_FG, TUI_DEFAULT_BG,
		statusPos, yPos)

	// count Zs and Bs and show that count in status
	var Zs, Bs int
	for _, up := range f.units {
		switch up.unit.(type) {
		case *Zed:
			Zs++
		case *Damsel:
			Bs++
		}
	}

	statusPos = writeTermString(fmt.Sprintf("Zs: %d", Zs), TUI_ZED_FG, TUI_DEFAULT_BG,
		statusPos+1, yPos)
	statusPos = writeTermString(fmt.Sprintf("Bs: %d", Bs), TUI_DAMSEL_FG, TUI_DEFAULT_BG,
		statusPos+1, yPos)

	// render gameover block if nesessary
	switch gameState {
	case GAME_WIN:
		writeBanner("YOU WIN")
	case GAME_LOSE:
		writeBanner("YOU LOSE")
	case GAME_DRAW:
		writeBanner("DRAW")
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

func writeTermString(str string, fg, bg termbox.Attribute, startX, startY int) (newPos int) {
	for _, r := range str {
		termbox.SetCell(startX, startY, r, fg, bg)
		startX++
	}
	return startX
}

func writeBanner(str string) {
	strlen := len(str)
	totalLen := strlen + 4

	size := tb2cell()
	xs := (size.X - totalLen) / 2
	ys := size.Y/2 -1

	msg := fmt.Sprintf("* %s *", str)
	line := strings.Repeat("*", totalLen)
	writeTermString(line, TUI_DEFAULT_FG, TUI_DEFAULT_BG, xs, ys)
	writeTermString(msg, TUI_DEFAULT_FG, TUI_DEFAULT_BG, xs, ys+1)
	writeTermString(line, TUI_DEFAULT_FG, TUI_DEFAULT_BG, xs, ys+2)
}
