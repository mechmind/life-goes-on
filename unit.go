package main

const (
	ZED_NUTRITION_WALKING         = 1
	ZED_NUTRITION_BITING          = 0.35
	ZED_RAGE_FROM_DAMAGE          = 1
	ZED_RAGE_COOLING              = 0.03
	ZED_RAGE_THRESHOLD            = 3
	ZED_RAGE_COST                 = 0.1
	ZED_RAGE_SPEEDUP              = 0.05
	ZED_RAGE_BITEUP               = 0.3
	ZED_NUTRITION_TO_HP_PORTION   = 5
	ZED_NUTRITION_TO_HP_THRESHOLD = 1200
	ZED_NUTRITION_TO_HP_SCALE     = 0.02
	ZED_NUTRITION_FULL            = 1600
	ZED_MOVER_WALK                = 0.65
	ZED_MOVER_WALKUP              = 0.4
	ZED_MOVER_WALKDOWN            = 0.6
	ZED_EAT_NUTRITION             = 350
	ZED_INFECT_NUTRITION          = 50
	ZED_BITE_DAMAGE               = 40
	ZED_HEALTH                    = 140
	ZED_NUTRITION_BASE            = 1000

	SOL_MOVER_WALK      = 0.70
	SOL_MOVER_WALKUP    = 0.25
	SOL_MOVER_WALKDOWN  = 0.75
	SOL_BASE_HEALTH     = 100
	SOL_GUN_DAMAGE      = 10
	SOL_GUN_RANGE       = 45
	SOL_ACC_DECAY_START = 10
	SOL_MISSHOT_PROB    = 20
	SOL_GREN_DAMAGE     = 80
	SOL_GREN_RANGE      = 20
	SOL_GREN_RADIUS     = 6
	SOL_GREN_SPEED      = 3
	SOL_GREN_TICK_CAP   = 3
	SOL_GREN_TIMEOUT    = 15
	SOL_SEMIFIRE_TICKS  = 2

	DAM_MOVER_WALK      = 0.30
	DAM_MOVER_WALKUP    = 0.10
	DAM_MOVER_WALKDOWN  = 0.35
	DAM_BASE_HEALTH     = 75
	DAM_SCREAM_RANGE    = 20
	DAM_PANIC_SPEEDUP   = 0.02
	DAM_PANIC_MAX_SPEED = 0.45
	DAM_ADRENALINE_FADE = 1
	DAM_FEAR_FACTOR     = 3

	CORPSE_RESSURECT_TICKS = 30
)

type Unit interface {
	SetID(int)
	GetID() int
	Mover
	DamageReciever
}

type Mover interface {
	MoveToward(src, dest UnitCoord) (UnitCoord, bool)
}

type DamageReciever interface {
	RecieveDamage(From int, dmg float32)
}

type Walker struct {
	WalkSpeed     float32
	WalkUpSpeed   float32
	WalkDownSpeed float32
}

func (w *Walker) MoveToward(f *Field, src, dest UnitCoord) (UnitCoord, bool) {
	toward := NormTowardCoord(src, dest)
	direction := NextCellCoord(src, toward)
	currentCellCoord := src.Cell()
	currentCell := f.CellAt(currentCellCoord)
	cost := calcSlopeCost(direction, currentCell.Slopes)

	//log.Println("mover:", src, "->", dest, "d:", direction, "t:", toward)
	var stuck bool
	var energy float32
	switch cost {
	case -1:
		energy = w.WalkDownSpeed
	case 0:
		energy = w.WalkSpeed
	case 1:
		energy = w.WalkUpSpeed
	}

	targetDistance := src.Distance(dest)
	if targetDistance < energy {
		energy = targetDistance
	}

	distance := toward.Mult(energy)
	next := src.AddCoord(distance)
	nextCellCoord := next.Cell()
	if nextCellCoord != currentCellCoord {
		// crossed bound, so check edge passability
		//log.Println("mover: crossing bounds", currentCellCoord, "->", nextCellCoord)
		stepCoord := currentCellCoord.AddCoord(direction)
		pass := f.CheckPassability(currentCellCoord, stepCoord)
		if pass == PS_PASSABLE {
			// ok moving in
			// check if have transit cell
			//log.Println("mover: can cross", currentCellCoord, "->", stepCoord)
			if stepCoord != nextCellCoord {
				// have a transit cell
				pass = f.CheckPassability(stepCoord, nextCellCoord)
				if pass != PS_PASSABLE {
					// stuck in transit cell
					// move into it and hang around edge
					// may help pathing algo
					var scale float32
					if direction.X != 0 {
						if direction.X > 0 {
							scale = (float32(stepCoord.X) - src.X) / distance.X
						} else {
							scale = (float32(currentCellCoord.X) - src.X) / distance.X
						}
					} else {
						if direction.Y > 0 {
							scale = (float32(stepCoord.Y) - src.Y) / distance.Y
						} else {
							scale = (float32(currentCellCoord.Y) - src.Y) / distance.Y
						}
					}
					scale -= FLOAT_ERROR
					next = src.AddCoord(distance.Mult(scale))
					stuck = true
				}
			}
		} else {
			// just hang there if cannot pass into next cell
			next = src
			stuck = true
			//log.Println("mover: shall not pass", currentCellCoord, "->", nextCellCoord)
		}
	}

	return next, stuck
}

func (w *Walker) MoveAway(f *Field, src, dest UnitCoord) (UnitCoord, bool) {
	newDest := src.AddCoord(src.AddCoord(dest.Mult(-1)))
	return w.MoveToward(f, src, newDest)
}

// calcSlopeCost return 1 for moving up on slope, -1 for moving down on slope
func calcSlopeCost(direction CellCoord, slope uint8) int {
	var cost int
	// check horizontal movement
	cost += (int((slope&SLOPE_DOWN)>>SLOPE_DOWN_SHIFT) -
		int((slope&SLOPE_UP)>>SLOPE_UP_SHIFT)) * direction.X
	cost += (int((slope&SLOPE_DOWN)>>SLOPE_DOWN_SHIFT) -
		int((slope&SLOPE_UP)>>SLOPE_UP_SHIFT)) * direction.Y
	return ibound(cost, -1, 1)
}

type Possesser struct {
	Items []Item
	Limit int
}

type Chaser struct {
	Target int
}

func (c *Chaser) LockOn(Target int) {
	c.Target = Target
}

type Gunner struct {
	FireRange float32
	GunDamage float32
}

func (g Gunner) CanShoot(src, dest UnitCoord) bool {
	return src.Distance(dest) < g.FireRange
}

type Biter struct {
	BiteDamage float32
}

func (b Biter) CanBite(src, dest UnitCoord) bool {
	return src.Distance(dest) < 1
}

type Soldier struct {
	Walker
	Possesser
	Chaser
	Gunner
	field           *Field
	Id              int
	Health          float32
	SemifireCounter int8
	Target          UnitCoord
	MyTarget        UnitCoord
	path            Path
}

func NewSoldier(field *Field) *Soldier {
	return &Soldier{Walker: Walker{SOL_MOVER_WALK, SOL_MOVER_WALKUP, SOL_MOVER_WALKDOWN},
		Chaser: Chaser{-1}, Gunner: Gunner{FireRange: SOL_GUN_RANGE, GunDamage: SOL_GUN_DAMAGE},
		Health: SOL_BASE_HEALTH, field: field}
}

func (s *Soldier) SetID(Id int) {
	s.Id = Id
}

func (s *Soldier) GetID() int {
	return s.Id
}

func (s *Soldier) MoveToward(src, dest UnitCoord) (UnitCoord, bool) {
	nextCoord, stuck := s.Walker.MoveToward(s.field, src, dest)
	return s.field.MoveMe(s.Id, nextCoord), stuck
}

func (s *Soldier) CanShoot(src, dest UnitCoord) bool {
	return s.Gunner.CanShoot(src, dest) && s.field.HaveLOS(src, dest) != VS_INVISIBLE
}

func (s *Soldier) RecieveDamage(From int, dmg float32) {
	s.Health -= dmg
	if s.Health < 0 {
		s.field.KillMe(s.Id)
	}
	s.Health -= dmg
}

func (s *Soldier) Shoot(src, dest UnitCoord, victim Unit) {
	// check misshots
	tid, newDest := s.field.TraceShot(src, dest, victim.GetID())
	// calculate hit probability
	dist := src.Distance(newDest)
	if dist > SOL_ACC_DECAY_START {
		prob := (SOL_GUN_RANGE - dist) * 100 / (SOL_GUN_RANGE - SOL_ACC_DECAY_START)
		if s.field.rng.Float32()*100 > prob {
			// miss
			return
		}
	}
	_, realVictim := s.field.UnitByID(tid)
	realVictim.RecieveDamage(s.Id, s.Gunner.GunDamage)
}

type Zed struct {
	field *Field
	Id    int

	Walker
	Chaser
	Biter
	Health       float32
	LastAttacker int

	Rage      float32
	Nutrition float32

	path Path
}

func NewZed(field *Field) *Zed {
	return &Zed{Walker: Walker{ZED_MOVER_WALK, ZED_MOVER_WALKUP, ZED_MOVER_WALKDOWN},
		Biter: Biter{BiteDamage: ZED_BITE_DAMAGE}, LastAttacker: -1, Rage: 0,
		Nutrition: ZED_NUTRITION_BASE, Health: ZED_HEALTH, field: field}
}

func (z *Zed) SetID(Id int) {
	z.Id = Id
}

func (z *Zed) GetID() int {
	return z.Id
}
func (z *Zed) MoveToward(src, dest UnitCoord) (UnitCoord, bool) {
	// apply nutrition and rage speedup/slowdown
	nutr_coeff := z.Nutrition / 1000
	rage_coeff := z.Rage * ZED_RAGE_SPEEDUP
	all_coeff := nutr_coeff + rage_coeff
	z.Walker = Walker{fbound(ZED_MOVER_WALK*all_coeff, 0, 1),
		fbound(ZED_MOVER_WALKUP*all_coeff, 0, 1),
		fbound(ZED_MOVER_WALKDOWN*all_coeff, 0, 1)}
	nextCoord, stuck := z.Walker.MoveToward(z.field, src, dest)
	z.Nutrition -= src.Distance(nextCoord) * ZED_NUTRITION_WALKING
	return z.field.MoveMe(z.Id, nextCoord), stuck
}

func (z *Zed) Bite(src, dest UnitCoord, victim Unit) {
	damage := z.Biter.BiteDamage + z.Rage*ZED_RAGE_BITEUP
	z.Nutrition -= damage * ZED_NUTRITION_BITING

	victim.RecieveDamage(z.Id, z.Biter.BiteDamage)
}

func (z *Zed) RecieveDamage(From int, dmg float32) {
	z.Health -= dmg
	z.Rage += dmg * ZED_RAGE_FROM_DAMAGE
	z.LastAttacker = From
	if z.Health < 0 {
		z.field.KillMe(z.Id)
	}
}

func (z *Zed) Eat(food float32) {
	z.Nutrition += food
}

func (z *Zed) Digest() bool {
	// calm down
	z.Rage -= z.Rage * ZED_RAGE_COOLING
	if z.Rage < ZED_RAGE_THRESHOLD {
		z.Rage = 0
	}

	// feed the anger
	z.Nutrition -= z.Rage * ZED_RAGE_COST

	// digest the food
	if z.Nutrition > ZED_NUTRITION_TO_HP_THRESHOLD {
		z.Nutrition -= ZED_NUTRITION_TO_HP_PORTION
		z.Health += ZED_NUTRITION_TO_HP_PORTION * ZED_NUTRITION_TO_HP_SCALE
	}

	if z.Nutrition < 0 {
		// starve to death
		z.field.KillMe(z.Id)
		return false
	}
	return true
}

type Damsel struct {
	Walker
	field        *Field
	Id           int
	Health       float32
	PanicPoint   UnitCoord
	Adrenaline   float32
	LastAttacker int
	WanderTarget UnitCoord
}

func NewDamsel(field *Field) *Damsel {
	return &Damsel{Walker: Walker{DAM_MOVER_WALK, DAM_MOVER_WALKUP, DAM_MOVER_WALKDOWN},
		LastAttacker: -1, Health: DAM_BASE_HEALTH, field: field}
}

func (d *Damsel) SetID(Id int) {
	d.Id = Id
}

func (d *Damsel) GetID() int {
	return d.Id
}

func (d *Damsel) MoveToward(src, dest UnitCoord) (UnitCoord, bool) {
	d.adjustWalkSpeed()
	nextCoord, stuck := d.Walker.MoveToward(d.field, src, dest)
	return d.field.MoveMe(d.Id, nextCoord), stuck
}

func (d *Damsel) MoveAway(src, dest UnitCoord) (UnitCoord, bool) {
	d.adjustWalkSpeed()
	nextCoord, stuck := d.Walker.MoveAway(d.field, src, dest)
	return d.field.MoveMe(d.Id, nextCoord), stuck
}

func (d *Damsel) adjustWalkSpeed() {
	// calculate adrenaline effect
	// FIXME: walkup/walkdown recalc
	newSpeed := DAM_MOVER_WALK + d.Adrenaline*DAM_PANIC_SPEEDUP
	d.Walker = Walker{fbound(newSpeed, 0, DAM_PANIC_MAX_SPEED),
		fbound(newSpeed, 0, DAM_PANIC_MAX_SPEED), fbound(newSpeed, 0, DAM_PANIC_MAX_SPEED)}
}

func (d *Damsel) HearScream(dmg float32, src UnitCoord, distance float32) {
	newAdrenaline := dmg / distance * DAM_FEAR_FACTOR
	if d.Adrenaline < FLOAT_ERROR {
		d.Adrenaline = newAdrenaline
	}

	d.PanicPoint = src
}

func (d *Damsel) RecieveDamage(From int, dmg float32) {
	d.Health -= dmg
	d.LastAttacker = From
	// scream in pain
	myCoord, _ := d.field.UnitByID(d.Id)
	neighs := d.field.UnitsInRange(myCoord, DAM_SCREAM_RANGE)
	for _, neigh := range neighs {
		if neighDam, ok := neigh.Unit.(*Damsel); ok && neighDam.Id != d.Id {
			neighDam.HearScream(dmg, myCoord, myCoord.Distance(neigh.Coord))
		}
	}

	if d.Health < 0 {
		d.field.KillMe(d.Id)
	} else {
		// boost adrenaline
		d.Adrenaline += dmg
	}
}

// temporary implement corpse as unit
type Corpse struct {
	field            *Field
	Id               int
	Unit             Unit
	RessurectCounter int
}

func (c *Corpse) SetID(Id int) {
	c.Id = Id
}

func (c *Corpse) GetID() int {
	return c.Id
}

func (c *Corpse) MoveToward(src, dest UnitCoord) (UnitCoord, bool) {
	return src, false
}

func (c *Corpse) RecieveDamage(From int, dmg float32) {}

func (c *Corpse) Respawn() *Zed {
	return NewZed(c.field)
}
