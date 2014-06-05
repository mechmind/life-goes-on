package main

import (
	"fmt"
	"github.com/nsf/termbox-go"
	"log"
	"strings"
)

const (
	TUI_DEFAULT_BG = termbox.ColorBlack
	TUI_DEFAULT_FG = termbox.ColorWhite

	TUI_SOLDIER_CHAR       = '@'
	TUI_SOLDIER_FG         = termbox.ColorRed | termbox.AttrBold
	TUI_ANOTHER_SOLDIER_FG = termbox.ColorMagenta | termbox.AttrBold

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
	TUI_STATUS_INFO_FG = termbox.ColorWhite | termbox.AttrBold

	MESSAGE_LEVEL_INFO = 1
	MESSAGE_LEVEL_RULE = 2
	MESSAGE_TTL        = 80
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

type Render interface {
	HandleUpdate(*Field)
	HandleGameState(GameState)
	HandleMessage(int, string)
	AssignSquad(int, chan Order)
	Spectate()
	Reset()
}

type Assignment struct {
	Id     int
	Orders chan Order
}

func (a Assignment) String() string {
	if a.Id >= 0 {
		if a.Orders != nil {
			return fmt.Sprintf("assgn{#%d with orders}", a.Id)
		} else {
			return fmt.Sprintf("assgn{#%d without orders}", a.Id)
		}
	} else {
		return fmt.Sprintf("assgn{spectator}")
	}
}

type Message struct {
	Level   int
	Content string
	ttl     int
}

type LocalRender struct {
	updates      chan *Field
	Orders       chan Order
	messages     chan Message
	squad        int
	stateUpdates chan GameState
	assignments  chan Assignment

	events chan termbox.Event
	reset  chan struct{}
}

func NewLocalRender() *LocalRender {
	return &LocalRender{updates: make(chan *Field, 3), stateUpdates: make(chan GameState, 3),
		squad: -1, assignments: make(chan Assignment, 1), events: make(chan termbox.Event),
		reset: make(chan struct{}, 1), messages: make(chan Message, 1)}
}

func (lr *LocalRender) HandleUpdate(f *Field) {
	select {
	case lr.updates <- f:
	default:
	}
}

func (lr *LocalRender) HandleGameState(s GameState) {
	select {
	case lr.stateUpdates <- s:
	default:
	}
}

func (lr *LocalRender) HandleMessage(lvl int, msg string) {
	select {
	case lr.messages <- Message{lvl, msg, MESSAGE_TTL}:
	default:
	}
}

func (lr *LocalRender) AssignSquad(Id int, Orders chan Order) {
	lr.assignments <- Assignment{Id, Orders}
}

func (lr *LocalRender) Spectate() {
	lr.assignments <- Assignment{-1, nil}
}

func (lr *LocalRender) Reset() {
	lr.reset <- struct{}{}
}

func (lr *LocalRender) Init() {
	go pollEvents(lr.events)

	prepareTerminal()

	termbox.Init()
	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)

}

func (lr *LocalRender) Run() {
	defer termbox.Close()

	log.Println("render: starting up")

	var currentPos CellCoord
	var doSquadFocus bool

	// FIXME: hardcoded squad values
	var sv = squadView{FireState: ORDER_FIRE}

	// recieve field view first
	log.Println("render: recieving very first field update")
	var field = <-lr.updates

	var gameState = GameState{State: GAME_WAIT}
	var rulesMsg string
	var msg Message

	lr.drawField(field, currentPos, sv, gameState, msg, rulesMsg)
	log.Println("render: starting main loop")
	for {
		select {
		case newMsg := <-lr.messages:
			if newMsg.Level == MESSAGE_LEVEL_RULE {
				rulesMsg = newMsg.Content
			} else {
				msg = newMsg
			}
			lr.drawField(field, currentPos, sv, gameState, msg, rulesMsg)
		case newGameState := <-lr.stateUpdates:
			if newGameState.State == GAME_OVER {
				gameState.State |= newGameState.State
			} else {
				gameState = newGameState
			}
			log.Println("render: game state changed to", gameState)
		case Assignment := <-lr.assignments:
			lr.squad = Assignment.Id
			lr.Orders = Assignment.Orders
			doSquadFocus = true
			log.Println("render: got new assignment:", Assignment)
		case <-lr.reset:
			sv = squadView{FireState: ORDER_FIRE}
			log.Println("render: resetting state")
		case field = <-lr.updates:

			// update rendering state
			// handle grens
			for _, gren := range field.Grens {
				if gren.From.Cell() == sv.GrenTo {
					sv.GrenTo = CellCoord{0, 0}
					break
				}
			}
			if doSquadFocus && lr.squad >= 0 {
				// center view on our squad
				for _, a := range field.Agents {
					if squad, ok := a.(*Squad); ok {
						if squad.Pid == lr.squad {
							if len(squad.Units) > 0 {
								centerOn, _ := field.UnitByID(squad.Units[0].Id)
								size := tb2cell()
								currentPos = centerOn.Cell().Add(-size.X/2, -size.Y/2)
								doSquadFocus = false
							}
							break
						}
					}
				}
			}
			lr.drawField(field, currentPos, sv, gameState, msg, rulesMsg)
		case ev := <-lr.events:
			switch ev.Type {
			case termbox.EventMouse:
				cursorPos := currentPos.Add(ev.MouseX, ev.MouseY)
				switch {
				case ev.Key == termbox.MouseLeft:
					if (lr.squad >= 0 &&
						CheckCellCoordBounds(cursorPos, CellCoord{0, 0}, CellCoord{1024, 1024}) &&
						field.CellAt(cursorPos).Passable) {
						sendOrder(lr.Orders, Order{ORDER_MOVE, cursorPos})
						sv.movingTo = cursorPos
					}
				case ev.Key == termbox.MouseRight:
					if lr.squad >= 0 {
						sendOrder(lr.Orders, Order{ORDER_GREN, cursorPos})
						sv.GrenTo = cursorPos
					}
				}

			case termbox.EventKey:
				switch {
				// direct moving window
				case ev.Key == termbox.KeyArrowLeft:
					fallthrough
				case ev.Ch == 'a':
					currentPos = currentPos.Add(-TUI_POS_STEP, 0)

				case ev.Key == termbox.KeyArrowRight:
					fallthrough
				case ev.Ch == 'd':
					currentPos = currentPos.Add(TUI_POS_STEP, 0)

				case ev.Key == termbox.KeyArrowDown:
					fallthrough
				case ev.Ch == 's':
					currentPos = currentPos.Add(0, TUI_POS_STEP)

				case ev.Key == termbox.KeyArrowUp:
					fallthrough
				case ev.Ch == 'w':
					currentPos = currentPos.Add(0, -TUI_POS_STEP)

				case ev.Ch == 'f':
					fallthrough
				case ev.Ch == 'F':
					if lr.squad >= 0 {
						sv.FireState = toggleFireState(sv.FireState)
						sendOrder(lr.Orders, Order{sv.FireState, CellCoord{0, 0}})
					}

				case ev.Ch == 'p':
					fallthrough
				case ev.Ch == 'P':
					if lr.squad >= 0 {
						sendOrder(lr.Orders, Order{ORDER_AUTOMOVE, CellCoord{0, 0}})
						sv.Automove = true
					}

				// quit
				case ev.Key == termbox.KeyF10:
					return
				}
				lr.drawField(field, currentPos, sv, gameState, msg, rulesMsg)
			case termbox.EventResize:
				lr.drawField(field, currentPos, sv, gameState, msg, rulesMsg)
			}
		}
		msg.ttl--
	}
}

// render field chunk that we currently looking at
func (lr *LocalRender) drawField(f *Field, pos CellCoord, sv squadView, gameState GameState,
	msg Message, rulesMsg string) {
	// 2 lines are reserved for messages and status bars
	upperBound := tb2cell().Add(-1, -3).AddCoord(pos)

	termbox.Clear(TUI_DEFAULT_FG, TUI_DEFAULT_BG)

	var fieldZero = CellCoord{0, 0}
	var fieldMax = CellCoord{f.XSize - 1, f.YSize - 1}
	// render walls
	for i := pos.X; i <= upperBound.X; i++ {
		for j := pos.Y; j <= upperBound.Y; j++ {
			tileCell := CellCoord{i, j}
			screenPos := tileCell.AddCoord(pos.Mult(-1))
			if !CheckCellCoordBounds(tileCell, fieldZero, fieldMax) {
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_OFFSCREEN_CHAR,
					TUI_DEFAULT_FG, TUI_DEFAULT_BG)
			} else {
				if f.CellAt(tileCell).Passable {
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
	for _, up := range f.Units {
		unitCell := up.Coord.Cell()
		if !CheckCellCoordBounds(unitCell, pos, upperBound) {
			// unit is not visible
			continue
		}
		ch, fg, bg := getUnitView(f, lr.squad, up.Unit)
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
	for _, gren := range f.Grens {
		if gren.Booming == 0 {
			// flying gren
			if CheckCellCoordBounds(gren.From.Cell(), pos, upperBound) {
				screenPos := gren.From.Cell().AddCoord(pos.Mult(-1))
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLYING_GREN_TARGET_CHAR,
					TUI_FLYING_GREN_TARGET_FG, TUI_DEFAULT_BG)
			}
		} else {
			// explosion
			center := gren.To
			for i := -SOL_GREN_RADIUS; i <= SOL_GREN_RADIUS; i++ {
				for j := -SOL_GREN_RADIUS; j <= SOL_GREN_RADIUS; j++ {
					cellCoord := center.Cell().Add(i, j)
					screenPos := cellCoord.AddCoord(pos.Mult(-1))
					if CheckCellCoordBounds(cellCoord, pos, upperBound) &&
						center.Distance(cellCoord.UnitCenter()) < SOL_GREN_RADIUS &&
						f.HaveLOS(center, cellCoord.UnitCenter()) {
						// in a range and visible
						boomingView := boomingColors[gren.Booming]
						termbox.SetCell(screenPos.X, screenPos.Y,
							boomingView.ch, boomingView.fg, boomingView.bg)
					}
				}
			}
		}
	}

	if (sv.GrenTo != CellCoord{0, 0}) && CheckCellCoordBounds(sv.GrenTo, pos, upperBound) {
		screenPos := sv.GrenTo.AddCoord(pos.Mult(-1))
		termbox.SetCell(screenPos.X, screenPos.Y, TUI_GREN_TARGET_CHAR,
			TUI_GREN_TARGET_FG, TUI_DEFAULT_BG)
	}
	// render pathfind
	// FIXME(pathfind)
	if p := f.pathfinder; p != nil {
		for Coord, cell := range p.Cells {
			break
			if !CheckCellCoordBounds(Coord, pos, upperBound) {
				continue
			}
			screenPos := Coord.AddCoord(pos.Mult(-1))
			switch {
			case cell.closed:
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
					TUI_DEFAULT_FG, TUI_PATHFIND_CLOSED_BG)
			case cell.open:
				termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
					TUI_DEFAULT_FG, TUI_PATHFIND_OPEN_BG)
			}
		}
		for _, Coord := range p.path {
			break
			if !CheckCellCoordBounds(Coord, pos, upperBound) {
				continue
			}
			screenPos := Coord.AddCoord(pos.Mult(-1))
			termbox.SetCell(screenPos.X, screenPos.Y, TUI_FLAT_CHAR,
				TUI_DEFAULT_FG, TUI_PATHFIND_PATH_BG)
		}
	}

	// render status and message bars
	var statusPos int
	yPos := tb2cell().Y - 1
	if lr.squad >= 0 {
		var FireState = "[ %s ]"
		switch sv.FireState {
		case ORDER_FIRE:
			FireState = fmt.Sprintf(FireState, "STAY_FIRE")
		case ORDER_SEMIFIRE:
			FireState = fmt.Sprintf(FireState, "RUN_FIRE")
		case ORDER_NOFIRE:
			FireState = fmt.Sprintf(FireState, "NO_FIRE")
		}
		statusPos = writeTermString(FireState, TUI_STATUS_FIRE_FG, TUI_DEFAULT_BG,
			statusPos, yPos)
	} else {
		// spectator mode
		statusPos = writeTermString("[spectator] ", TUI_STATUS_INFO_FG, TUI_DEFAULT_BG,
			statusPos, yPos)
	}

	// render message, if any
	if msg.ttl > 0 {
		switch msg.Level {
		case MESSAGE_LEVEL_INFO:
			writeTermString(msg.Content, TUI_STATUS_INFO_FG, TUI_DEFAULT_BG, 0, yPos-1)
		}
	}

	// count Zs and Bs and show that count in status
	var Zs, Bs int
	for _, up := range f.Units {
		switch up.Unit.(type) {
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

	statusPos = writeTermString(rulesMsg, TUI_STATUS_INFO_FG, TUI_DEFAULT_BG,
		statusPos+1, yPos)

	// render gameover block if nesessary
	var banner string
	switch {
	case gameState.State&GAME_WIN > 0:
		banner = "YOU WIN"
	case gameState.State&GAME_LOSE > 0:
		banner = "YOU LOSE"
	case gameState.State&GAME_DRAW > 0:
		banner = "DRAW"
	}
	if gameState.State&GAME_OVER > 0 {
		if banner == "" {
			banner = "GAME OVER"
		} else {
			banner += " | GAME OVER"
		}
	}
	if banner != "" {
		writeBanner(banner)
	}
	termbox.Flush()
}

func getUnitView(f *Field, pid int, u Unit) (ch rune, fg, bg termbox.Attribute) {
	switch u.(type) {
	case *Soldier:
		var solColor termbox.Attribute = TUI_ANOTHER_SOLDIER_FG
		s := u.(*Soldier)
		agent := f.AgentForUnitID(s.Id)

		if squad, ok := agent.(*Squad); ok {
			if squad.Pid == pid {
				solColor = TUI_SOLDIER_FG
			}
		}
		return TUI_SOLDIER_CHAR, solColor, TUI_DEFAULT_BG
	case *Damsel:
		dam := u.(*Damsel)
		if dam.Adrenaline > 0 {
			return TUI_FASTDAMSEL_CHAR, TUI_FASTDAMSEL_FG, TUI_DEFAULT_BG
		} else {
			return TUI_DAMSEL_CHAR, TUI_DAMSEL_FG, TUI_DEFAULT_BG
		}
	case *Zed:
		zed := u.(*Zed)
		if zed.Nutrition > ZED_NUTRITION_FULL {
			return TUI_FASTZED_CHAR, TUI_FASTZED_FG, TUI_DEFAULT_BG
		} else {
			return TUI_ZED_CHAR, TUI_ZED_FG, TUI_DEFAULT_BG
		}
	case *Corpse:
		corpse := u.(*Corpse)
		ch, fg, _ := getUnitView(f, pid, corpse.Unit)
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
	ys := size.Y/2 - 1

	msg := fmt.Sprintf("* %s *", str)
	line := strings.Repeat("*", totalLen)
	writeTermString(line, TUI_DEFAULT_FG, TUI_DEFAULT_BG, xs, ys)
	writeTermString(msg, TUI_DEFAULT_FG, TUI_DEFAULT_BG, xs, ys+1)
	writeTermString(line, TUI_DEFAULT_FG, TUI_DEFAULT_BG, xs, ys+2)
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

func sendOrder(Orders chan Order, o Order) {
	select {
	case Orders <- o:
	default:
	}
}

type squadView struct {
	FireState int
	movingTo  CellCoord
	GrenTo    CellCoord
	Automove  bool
}
