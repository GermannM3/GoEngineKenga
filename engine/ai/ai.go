package ai

import (
	"container/heap"
	"math"

	emath "goenginekenga/engine/math"
)

// Pathfinding using A*

// NavGrid represents a navigation grid
type NavGrid struct {
	Width    int
	Height   int
	Walkable []bool
	Costs    []float32 // Movement cost for each cell (1.0 = normal)
}

// NewNavGrid creates a new navigation grid
func NewNavGrid(width, height int) *NavGrid {
	size := width * height
	grid := &NavGrid{
		Width:    width,
		Height:   height,
		Walkable: make([]bool, size),
		Costs:    make([]float32, size),
	}
	// Initialize all cells as walkable with cost 1
	for i := range grid.Walkable {
		grid.Walkable[i] = true
		grid.Costs[i] = 1.0
	}
	return grid
}

// SetWalkable sets whether a cell is walkable
func (g *NavGrid) SetWalkable(x, y int, walkable bool) {
	if x >= 0 && x < g.Width && y >= 0 && y < g.Height {
		g.Walkable[y*g.Width+x] = walkable
	}
}

// IsWalkable returns whether a cell is walkable
func (g *NavGrid) IsWalkable(x, y int) bool {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return false
	}
	return g.Walkable[y*g.Width+x]
}

// SetCost sets the movement cost of a cell
func (g *NavGrid) SetCost(x, y int, cost float32) {
	if x >= 0 && x < g.Width && y >= 0 && y < g.Height {
		g.Costs[y*g.Width+x] = cost
	}
}

// GetCost returns the movement cost of a cell
func (g *NavGrid) GetCost(x, y int) float32 {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return math.MaxFloat32
	}
	return g.Costs[y*g.Width+x]
}

// PathNode represents a node in the path
type PathNode struct {
	X, Y int
}

// FindPath finds a path using A*
func (g *NavGrid) FindPath(startX, startY, endX, endY int) []PathNode {
	if !g.IsWalkable(startX, startY) || !g.IsWalkable(endX, endY) {
		return nil
	}

	// Priority queue implementation
	pq := &nodeHeap{}
	heap.Init(pq)

	// Maps for tracking visited nodes
	openSet := make(map[int]*astarNode)
	closedSet := make(map[int]bool)

	nodeKey := func(x, y int) int { return y*g.Width + x }

	// Heuristic (Manhattan distance)
	heuristic := func(x, y int) float32 {
		dx := float32(abs(x - endX))
		dy := float32(abs(y - endY))
		return dx + dy
	}

	// Start node
	startNode := &astarNode{
		x: startX,
		y: startY,
		g: 0,
		h: heuristic(startX, startY),
	}
	startNode.f = startNode.g + startNode.h
	heap.Push(pq, startNode)
	openSet[nodeKey(startX, startY)] = startNode

	// Direction offsets (8-directional movement)
	dirs := [][2]int{
		{0, -1}, {1, 0}, {0, 1}, {-1, 0}, // Cardinal
		{1, -1}, {1, 1}, {-1, 1}, {-1, -1}, // Diagonal
	}
	dirCosts := []float32{1, 1, 1, 1, 1.414, 1.414, 1.414, 1.414}

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*astarNode)
		currentKey := nodeKey(current.x, current.y)

		// Check if we reached the goal
		if current.x == endX && current.y == endY {
			// Reconstruct path
			var path []PathNode
			for node := current; node != nil; node = node.parent {
				path = append([]PathNode{{X: node.x, Y: node.y}}, path...)
			}
			return path
		}

		closedSet[currentKey] = true
		delete(openSet, currentKey)

		// Check neighbors
		for i, dir := range dirs {
			nx, ny := current.x+dir[0], current.y+dir[1]

			if !g.IsWalkable(nx, ny) {
				continue
			}

			neighborKey := nodeKey(nx, ny)
			if closedSet[neighborKey] {
				continue
			}

			// Calculate cost
			moveCost := dirCosts[i] * g.GetCost(nx, ny)
			tentativeG := current.g + moveCost

			neighbor, inOpen := openSet[neighborKey]
			if !inOpen {
				neighbor = &astarNode{
					x: nx,
					y: ny,
					h: heuristic(nx, ny),
				}
			} else if tentativeG >= neighbor.g {
				continue
			}

			neighbor.g = tentativeG
			neighbor.f = neighbor.g + neighbor.h
			neighbor.parent = current

			if !inOpen {
				heap.Push(pq, neighbor)
				openSet[neighborKey] = neighbor
			} else {
				heap.Fix(pq, neighbor.index)
			}
		}
	}

	return nil // No path found
}

// astarNode for heap
type astarNode struct {
	x, y   int
	g      float32
	h      float32
	f      float32
	parent *astarNode
	index  int
}

// Heap implementation for A*
type nodeHeap []*astarNode

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(i, j int) bool { return h[i].f < h[j].f }
func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *nodeHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*astarNode)
	item.index = n
	*h = append(*h, item)
}

func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Behavior Trees

// NodeStatus represents the status of a behavior node
type NodeStatus int

const (
	StatusRunning NodeStatus = iota
	StatusSuccess
	StatusFailure
)

// BehaviorNode is the interface for all behavior tree nodes
type BehaviorNode interface {
	Execute(agent *Agent) NodeStatus
	Reset()
}

// Agent represents an AI agent
type Agent struct {
	Position  emath.Vec3
	Target    emath.Vec3
	Speed     float32
	State     string
	Memory    map[string]interface{}
	Path      []PathNode
	PathIndex int
	NavGrid   *NavGrid
}

// NewAgent creates a new AI agent
func NewAgent(pos emath.Vec3, speed float32) *Agent {
	return &Agent{
		Position: pos,
		Speed:    speed,
		Memory:   make(map[string]interface{}),
	}
}

// MoveTo sets a movement target
func (a *Agent) MoveTo(target emath.Vec3) {
	a.Target = target
	if a.NavGrid != nil {
		// Find path
		a.Path = a.NavGrid.FindPath(
			int(a.Position.X), int(a.Position.Z),
			int(target.X), int(target.Z),
		)
		a.PathIndex = 0
	}
}

// Update updates the agent
func (a *Agent) Update(dt float32) {
	if len(a.Path) == 0 || a.PathIndex >= len(a.Path) {
		return
	}

	// Get current waypoint
	waypoint := a.Path[a.PathIndex]
	targetPos := emath.Vec3{X: float32(waypoint.X), Y: a.Position.Y, Z: float32(waypoint.Y)}

	// Move towards waypoint
	dir := emath.Vec3{
		X: targetPos.X - a.Position.X,
		Y: 0,
		Z: targetPos.Z - a.Position.Z,
	}

	dist := float32(math.Sqrt(float64(dir.X*dir.X + dir.Z*dir.Z)))
	if dist < 0.5 {
		// Reached waypoint
		a.PathIndex++
	} else {
		// Move
		dir.X /= dist
		dir.Z /= dist
		a.Position.X += dir.X * a.Speed * dt
		a.Position.Z += dir.Z * a.Speed * dt
	}
}

// Behavior tree nodes

// SequenceNode executes children in order until one fails
type SequenceNode struct {
	Children     []BehaviorNode
	currentChild int
}

func (n *SequenceNode) Execute(agent *Agent) NodeStatus {
	for n.currentChild < len(n.Children) {
		status := n.Children[n.currentChild].Execute(agent)
		if status == StatusRunning {
			return StatusRunning
		}
		if status == StatusFailure {
			n.currentChild = 0
			return StatusFailure
		}
		n.currentChild++
	}
	n.currentChild = 0
	return StatusSuccess
}

func (n *SequenceNode) Reset() {
	n.currentChild = 0
	for _, child := range n.Children {
		child.Reset()
	}
}

// SelectorNode executes children until one succeeds
type SelectorNode struct {
	Children     []BehaviorNode
	currentChild int
}

func (n *SelectorNode) Execute(agent *Agent) NodeStatus {
	for n.currentChild < len(n.Children) {
		status := n.Children[n.currentChild].Execute(agent)
		if status == StatusRunning {
			return StatusRunning
		}
		if status == StatusSuccess {
			n.currentChild = 0
			return StatusSuccess
		}
		n.currentChild++
	}
	n.currentChild = 0
	return StatusFailure
}

func (n *SelectorNode) Reset() {
	n.currentChild = 0
	for _, child := range n.Children {
		child.Reset()
	}
}

// ConditionNode checks a condition
type ConditionNode struct {
	Check func(agent *Agent) bool
}

func (n *ConditionNode) Execute(agent *Agent) NodeStatus {
	if n.Check(agent) {
		return StatusSuccess
	}
	return StatusFailure
}

func (n *ConditionNode) Reset() {}

// ActionNode performs an action
type ActionNode struct {
	Action func(agent *Agent) NodeStatus
}

func (n *ActionNode) Execute(agent *Agent) NodeStatus {
	return n.Action(agent)
}

func (n *ActionNode) Reset() {}

// WaitNode waits for a duration
type WaitNode struct {
	Duration float32
	elapsed  float32
}

func (n *WaitNode) Execute(agent *Agent) NodeStatus {
	n.elapsed += 0.016 // Assume 60fps
	if n.elapsed >= n.Duration {
		return StatusSuccess
	}
	return StatusRunning
}

func (n *WaitNode) Reset() {
	n.elapsed = 0
}

// RepeatNode repeats a child node
type RepeatNode struct {
	Child BehaviorNode
	Times int // 0 for infinite
	count int
}

func (n *RepeatNode) Execute(agent *Agent) NodeStatus {
	status := n.Child.Execute(agent)
	if status == StatusRunning {
		return StatusRunning
	}

	n.Child.Reset()
	n.count++

	if n.Times > 0 && n.count >= n.Times {
		n.count = 0
		return StatusSuccess
	}
	return StatusRunning
}

func (n *RepeatNode) Reset() {
	n.count = 0
	n.Child.Reset()
}

// State Machine

// State represents a state in a finite state machine
type State struct {
	Name        string
	OnEnter     func(agent *Agent)
	OnUpdate    func(agent *Agent, dt float32)
	OnExit      func(agent *Agent)
	Transitions []Transition
}

// Transition represents a state transition
type Transition struct {
	Condition func(agent *Agent) bool
	NextState string
}

// StateMachine manages state transitions
type StateMachine struct {
	States       map[string]*State
	CurrentState *State
	agent        *Agent
}

// NewStateMachine creates a new state machine
func NewStateMachine(agent *Agent) *StateMachine {
	return &StateMachine{
		States: make(map[string]*State),
		agent:  agent,
	}
}

// AddState adds a state
func (sm *StateMachine) AddState(state *State) {
	sm.States[state.Name] = state
}

// SetState sets the current state
func (sm *StateMachine) SetState(name string) {
	newState := sm.States[name]
	if newState == nil {
		return
	}

	if sm.CurrentState != nil && sm.CurrentState.OnExit != nil {
		sm.CurrentState.OnExit(sm.agent)
	}

	sm.CurrentState = newState
	if sm.CurrentState.OnEnter != nil {
		sm.CurrentState.OnEnter(sm.agent)
	}
}

// Update updates the state machine
func (sm *StateMachine) Update(dt float32) {
	if sm.CurrentState == nil {
		return
	}

	// Check transitions
	for _, trans := range sm.CurrentState.Transitions {
		if trans.Condition(sm.agent) {
			sm.SetState(trans.NextState)
			return
		}
	}

	// Update current state
	if sm.CurrentState.OnUpdate != nil {
		sm.CurrentState.OnUpdate(sm.agent, dt)
	}
}

// Preset AI behaviors

// CreatePatrolBehavior creates a patrol behavior tree
func CreatePatrolBehavior(waypoints []emath.Vec3) BehaviorNode {
	waypointIndex := 0

	return &RepeatNode{
		Times: 0, // Infinite
		Child: &SequenceNode{
			Children: []BehaviorNode{
				&ActionNode{
					Action: func(agent *Agent) NodeStatus {
						if waypointIndex >= len(waypoints) {
							waypointIndex = 0
						}
						agent.MoveTo(waypoints[waypointIndex])
						waypointIndex++
						return StatusSuccess
					},
				},
				&ActionNode{
					Action: func(agent *Agent) NodeStatus {
						if agent.PathIndex >= len(agent.Path) {
							return StatusSuccess
						}
						return StatusRunning
					},
				},
				&WaitNode{Duration: 2.0},
			},
		},
	}
}

// CreateChaseBehavior creates a chase behavior
func CreateChaseBehavior(getTargetPos func() emath.Vec3, chaseRange float32) BehaviorNode {
	return &SequenceNode{
		Children: []BehaviorNode{
			&ConditionNode{
				Check: func(agent *Agent) bool {
					target := getTargetPos()
					dx := target.X - agent.Position.X
					dz := target.Z - agent.Position.Z
					dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
					return dist <= chaseRange
				},
			},
			&ActionNode{
				Action: func(agent *Agent) NodeStatus {
					agent.MoveTo(getTargetPos())
					return StatusSuccess
				},
			},
		},
	}
}
