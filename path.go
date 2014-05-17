package main

import (
	"math"
)

var (
	neighbours = [8]CellCoord{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}}
)

type PathFinder struct {
	source, target CellCoord
	field          *Field
	cells          map[CellCoord]PathCell
	open           *WeightedList
	path Path
}

func NewPathFinder(f *Field) *PathFinder {
	return &PathFinder{field: f, cells: make(map[CellCoord]PathCell), open: &WeightedList{}}
}

func (p *PathFinder) FindPath(from, to CellCoord) Path {
	p.source = from
	p.target = to

	// initialize algo
	pc := PathCell{from, from, 0, true, true, false}
	p.open.Insert(from, from.Distance(to))
	p.cells[from] = pc

	// run algo
	return p.findPath()
}

func (p *PathFinder) CellAt(coord CellCoord) PathCell {
	cell, ok := p.cells[coord]
	if !ok {
		cell = PathCell{coord, CellCoord{}, math.MaxFloat32, false, false, false}
		if p.field.CellAt(coord).passable {
			cell.visible = true
		}
		p.cells[coord] = cell
	}

	return cell
}

func (p *PathFinder) closeCell(coord CellCoord) {
	pc := p.cells[coord]
	pc.closed = true
	p.cells[coord] = pc
	p.open.Remove(coord)
}

func (p *PathFinder) openCell(coord CellCoord, weight float32) {
	pc := p.cells[coord]
	pc.open = true
	p.cells[coord] = pc
	p.open.Insert(coord, weight)
}

func (p *PathFinder) updateCell(cell PathCell) {
	oldCell := p.cells[cell.coord]
	weight := cell.cost + cell.coord.Distance(p.target)
	if oldCell.open {
		p.open.Replace(cell.coord, weight)
	} else {
		p.open.Insert(cell.coord, weight)
	}
	// restore visibility
	cell.visible = oldCell.visible
	p.cells[cell.coord] = cell
}

func (p *PathFinder) Neighbours(center CellCoord) []PathCell {
	cells := make([]PathCell, 8)
	for idx, delta := range neighbours {
		cells[idx] = p.CellAt(center.AddCoord(delta))
	}
	return cells
}

func (p *PathFinder) findPath() Path {
	// just a*
	for {
		coord, ok := p.open.Pop()
		if !ok {
			// no more cells, no way to target
			return nil
		}

		if coord == p.target {
			// ok, path found

			path := p.backtrackPath()
			p.path = path
			return path
		}

		// close cell
		p.closeCell(coord)
		cell := p.CellAt(coord)

		// check neighbours
		neighbours := p.Neighbours(coord)
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
					neighbours[idx].parent = coord
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
	path := Path{p.target}
	curr := p.cells[p.target]
	for {
		curr = p.cells[curr.parent]
		if curr.coord == p.source {
			return path
		}
		path = append(path, curr.coord)
	}
}

func setVisibility(neighbours []PathCell) {
	// diagonal cells are not passable if adjacent edge cells are impassable
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

func (p *Path) Next() (coord CellCoord, ok bool) {
	path := *p
	if len(path) > 1 {
		path = path[:len(path)-1]
		coord = path[len(path)-1]
		*p = path
		return coord, true
	} else if len(path) == 1 {
		coord = path[0]
		*p = nil
		return coord, true
	} else {
		return CellCoord{}, false
	}
}

func (p *Path) Current() (coord CellCoord, ok bool) {
	if len(*p) > 0 {
		return (*p)[len(*p)-1], true
	} else {
		return CellCoord{}, false
	}
}

type PathCell struct {
	coord, parent CellCoord
	cost          float32
	visible       bool
	open, closed  bool
}

type WeightedList struct {
	head *WeightedCell
}

func (w *WeightedList) Insert(coord CellCoord, weight float32) {
	wc := &WeightedCell{coord, weight, nil}
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

func (w *WeightedList) Replace(coord CellCoord, weight float32) {
	// TODO: optimize
	w.Remove(coord)
	w.Insert(coord, weight)
}

func (w *WeightedList) Remove(coord CellCoord) {
	curr := w.head

	if curr == nil {
		return
	}

	if curr.coord == coord {
		w.head = curr.next
		return
	}

	next := curr.next

	for {
		if next == nil {
			// cell not found
			return
		}

		if next.coord == coord {
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
	return cell.coord, true
}

type WeightedCell struct {
	coord  CellCoord
	weight float32
	next   *WeightedCell
}
