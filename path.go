package main

import (
	"math"
)

var (
	neighbours = [8]CellCoord{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}}
)

type PathFinder struct {
	source, Target CellCoord
	field          *Field
	Cells          map[CellCoord]PathCell
	open           *WeightedList
	path           Path
}

func NewPathFinder(f *Field) *PathFinder {
	return &PathFinder{field: f, Cells: make(map[CellCoord]PathCell), open: &WeightedList{}}
}

func (p *PathFinder) FindPath(From, To CellCoord) Path {
	p.source = From
	p.Target = To

	// initialize algo
	pc := PathCell{From, From, 0, true, true, false}
	p.open.Insert(From, From.Distance(To))
	p.Cells[From] = pc

	// run algo
	return p.findPath()
}

func (p *PathFinder) CellAt(Coord CellCoord) PathCell {
	cell, ok := p.Cells[Coord]
	if !ok {
		cell = PathCell{Coord, CellCoord{}, math.MaxFloat32, false, false, false}
		if p.field.CellAt(Coord).Passable {
			cell.visible = true
		}
		p.Cells[Coord] = cell
	}

	return cell
}

func (p *PathFinder) closeCell(Coord CellCoord) {
	pc := p.Cells[Coord]
	pc.closed = true
	p.Cells[Coord] = pc
	p.open.Remove(Coord)
}

func (p *PathFinder) openCell(Coord CellCoord, weight float32) {
	pc := p.Cells[Coord]
	pc.open = true
	p.Cells[Coord] = pc
	p.open.Insert(Coord, weight)
}

func (p *PathFinder) updateCell(cell PathCell) {
	oldCell := p.Cells[cell.Coord]
	weight := cell.cost + cell.Coord.Distance(p.Target)
	if oldCell.open {
		p.open.Replace(cell.Coord, weight)
	} else {
		p.open.Insert(cell.Coord, weight)
	}
	// restore visibility
	cell.visible = oldCell.visible
	p.Cells[cell.Coord] = cell
}

func (p *PathFinder) Neighbours(center CellCoord) []PathCell {
	Cells := make([]PathCell, 8)
	for idx, delta := range neighbours {
		Cells[idx] = p.CellAt(center.AddCoord(delta))
	}
	return Cells
}

func (p *PathFinder) findPath() Path {
	// just a*
	for {
		Coord, ok := p.open.Pop()
		if !ok {
			// no more cells, no way to target
			return nil
		}

		if Coord == p.Target {
			// ok, path found

			path := p.backtrackPath()
			p.path = path
			return path
		}

		// close cell
		p.closeCell(Coord)
		cell := p.CellAt(Coord)

		// check neighbours
		neighbours := p.Neighbours(Coord)
		setVisibility(neighbours)

		var newCost float32
		for idx := range neighbours {
			if neighbours[idx].visible {
				if neighbours[idx].closed {
					// cell already expanded
					continue
				}
				// compute cost for pc and place/update it into open list
				if idx%2 == 0 {
					newCost = cell.cost + 1
				} else {
					newCost = cell.cost + math.Sqrt2
				}

				if newCost < neighbours[idx].cost {
					// update path for pc
					neighbours[idx].parent = Coord
					neighbours[idx].cost = newCost
					neighbours[idx].open = true
					p.updateCell(neighbours[idx])
				}
			}
		}
	}
	return nil
}

func (p *PathFinder) backtrackPath() Path {
	path := Path{p.Target}
	curr := p.Cells[p.Target]
	for {
		curr = p.Cells[curr.parent]
		if curr.Coord == p.source {
			return path
		}
		path = append(path, curr.Coord)
	}
}

func setVisibility(neighbours []PathCell) {
	// diagonal cells are not passable if adjacent edge cells are impassable
	// TODO: use Field.CheckPassability
	if !neighbours[0].visible {
		neighbours[1].visible = false
		neighbours[7].visible = false
	}
	if !neighbours[2].visible {
		neighbours[1].visible = false
		neighbours[3].visible = false
	}
	if !neighbours[4].visible {
		neighbours[3].visible = false
		neighbours[5].visible = false
	}
	if !neighbours[6].visible {
		neighbours[5].visible = false
		neighbours[7].visible = false
	}
}

type Path []CellCoord

func (p *Path) Next() (Coord CellCoord, ok bool) {
	path := *p
	if len(path) > 1 {
		path = path[:len(path)-1]
		Coord = path[len(path)-1]
		*p = path
		return Coord, true
	} else if len(path) == 1 {
		Coord = path[0]
		*p = nil
		return Coord, true
	} else {
		return CellCoord{}, false
	}
}

func (p *Path) Current() (Coord CellCoord, ok bool) {
	if len(*p) > 0 {
		return (*p)[len(*p)-1], true
	} else {
		return CellCoord{}, false
	}
}

type PathCell struct {
	Coord, parent CellCoord
	cost          float32
	visible       bool
	open, closed  bool
}

type WeightedList struct {
	head *WeightedCell
}

func (w *WeightedList) Insert(Coord CellCoord, weight float32) {
	wc := &WeightedCell{Coord, weight, nil}
	if w.head == nil {
		w.head = wc
		return
	}

	if w.head.weight > wc.weight {
		// replace head cell
		wc.next = w.head
		w.head = wc
		return
	}

	curr := w.head
	next := w.head.next
	for {
		// reached tail
		if next == nil {
			curr.next = wc
			return
		}

		// found right place
		if next.weight > wc.weight {
			curr.next, wc.next = wc, next
			return
		}

		// else step further
		curr, next = next, next.next
	}
}

func (w *WeightedList) Replace(Coord CellCoord, weight float32) {
	// TODO: optimize
	w.Remove(Coord)
	w.Insert(Coord, weight)
}

func (w *WeightedList) Remove(Coord CellCoord) {
	curr := w.head

	if curr == nil {
		return
	}

	if curr.Coord == Coord {
		w.head = curr.next
		return
	}

	next := curr.next

	for {
		if next == nil {
			// cell not found
			return
		}

		if next.Coord == Coord {
			// cell found
			curr.next = next.next
			return
		}
		// traverse deeper
		curr, next = next, next.next
	}
}

func (w *WeightedList) Pop() (CellCoord, bool) {
	if w.head == nil {
		return CellCoord{}, false
	}

	var cell *WeightedCell
	cell, w.head = w.head, w.head.next
	return cell.Coord, true
}

type WeightedCell struct {
	Coord  CellCoord
	weight float32
	next   *WeightedCell
}
