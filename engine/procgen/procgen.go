package procgen

import (
	"math"
	"math/rand"
)

// Noise generators and procedural generation utilities

// Noise2D generates 2D Perlin-like noise
type Noise2D struct {
	perm []int
	seed int64
}

// NewNoise2D creates a new 2D noise generator
func NewNoise2D(seed int64) *Noise2D {
	n := &Noise2D{
		perm: make([]int, 512),
		seed: seed,
	}
	n.reseed(seed)
	return n
}

func (n *Noise2D) reseed(seed int64) {
	r := rand.New(rand.NewSource(seed))
	p := make([]int, 256)
	for i := range p {
		p[i] = i
	}
	r.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })
	for i := 0; i < 256; i++ {
		n.perm[i] = p[i]
		n.perm[i+256] = p[i]
	}
}

// Sample returns noise value at (x, y), range [-1, 1]
func (n *Noise2D) Sample(x, y float64) float64 {
	// Find unit square containing point
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255

	// Relative position in square
	x -= math.Floor(x)
	y -= math.Floor(y)

	// Compute fade curves
	u := fade(x)
	v := fade(y)

	// Hash corners of square
	A := n.perm[X] + Y
	B := n.perm[X+1] + Y

	// Bilinear interpolation
	return lerp(v,
		lerp(u, grad2(n.perm[A], x, y), grad2(n.perm[B], x-1, y)),
		lerp(u, grad2(n.perm[A+1], x, y-1), grad2(n.perm[B+1], x-1, y-1)))
}

// FBM generates Fractal Brownian Motion (multiple octaves of noise)
func (n *Noise2D) FBM(x, y float64, octaves int, persistence float64) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0
	maxValue := 0.0

	for i := 0; i < octaves; i++ {
		total += n.Sample(x*frequency, y*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= 2
	}

	return total / maxValue
}

// Turbulence generates turbulence (absolute value of noise)
func (n *Noise2D) Turbulence(x, y float64, octaves int) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0

	for i := 0; i < octaves; i++ {
		total += math.Abs(n.Sample(x*frequency, y*frequency)) * amplitude
		amplitude *= 0.5
		frequency *= 2
	}

	return total
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

func grad2(hash int, x, y float64) float64 {
	h := hash & 3
	switch h {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	default:
		return -x - y
	}
}

// Heightmap generation

// Heightmap represents a 2D heightmap
type Heightmap struct {
	Width  int
	Height int
	Data   []float32
}

// NewHeightmap creates a new heightmap
func NewHeightmap(width, height int) *Heightmap {
	return &Heightmap{
		Width:  width,
		Height: height,
		Data:   make([]float32, width*height),
	}
}

// Get returns height at (x, y)
func (h *Heightmap) Get(x, y int) float32 {
	if x < 0 || x >= h.Width || y < 0 || y >= h.Height {
		return 0
	}
	return h.Data[y*h.Width+x]
}

// Set sets height at (x, y)
func (h *Heightmap) Set(x, y int, value float32) {
	if x >= 0 && x < h.Width && y >= 0 && y < h.Height {
		h.Data[y*h.Width+x] = value
	}
}

// GetInterpolated returns interpolated height
func (h *Heightmap) GetInterpolated(x, y float32) float32 {
	x0 := int(math.Floor(float64(x)))
	y0 := int(math.Floor(float64(y)))
	fx := x - float32(x0)
	fy := y - float32(y0)

	h00 := h.Get(x0, y0)
	h10 := h.Get(x0+1, y0)
	h01 := h.Get(x0, y0+1)
	h11 := h.Get(x0+1, y0+1)

	h0 := h00*(1-fx) + h10*fx
	h1 := h01*(1-fx) + h11*fx

	return h0*(1-fy) + h1*fy
}

// GenerateFromNoise fills heightmap using noise
func (h *Heightmap) GenerateFromNoise(noise *Noise2D, scale, amplitude float64, octaves int) {
	for y := 0; y < h.Height; y++ {
		for x := 0; x < h.Width; x++ {
			nx := float64(x) / float64(h.Width) * scale
			ny := float64(y) / float64(h.Height) * scale
			value := noise.FBM(nx, ny, octaves, 0.5)
			h.Data[y*h.Width+x] = float32((value + 1) * 0.5 * amplitude)
		}
	}
}

// Normalize normalizes heightmap to [0, 1] range
func (h *Heightmap) Normalize() {
	min, max := h.Data[0], h.Data[0]
	for _, v := range h.Data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if max-min < 0.0001 {
		return
	}

	scale := 1.0 / (max - min)
	for i := range h.Data {
		h.Data[i] = (h.Data[i] - min) * scale
	}
}

// Dungeon generation

// DungeonTile represents a tile in a dungeon
type DungeonTile int

const (
	TileWall DungeonTile = iota
	TileFloor
	TileDoor
	TileStairsUp
	TileStairsDown
	TileWater
	TileChest
)

// Room represents a dungeon room
type Room struct {
	X, Y, W, H int
	Connected  bool
}

// Dungeon represents a generated dungeon
type Dungeon struct {
	Width  int
	Height int
	Tiles  []DungeonTile
	Rooms  []Room
}

// NewDungeon creates a new dungeon
func NewDungeon(width, height int) *Dungeon {
	d := &Dungeon{
		Width:  width,
		Height: height,
		Tiles:  make([]DungeonTile, width*height),
	}
	// Fill with walls
	for i := range d.Tiles {
		d.Tiles[i] = TileWall
	}
	return d
}

// Get returns tile at (x, y)
func (d *Dungeon) Get(x, y int) DungeonTile {
	if x < 0 || x >= d.Width || y < 0 || y >= d.Height {
		return TileWall
	}
	return d.Tiles[y*d.Width+x]
}

// Set sets tile at (x, y)
func (d *Dungeon) Set(x, y int, tile DungeonTile) {
	if x >= 0 && x < d.Width && y >= 0 && y < d.Height {
		d.Tiles[y*d.Width+x] = tile
	}
}

// GenerateBSP generates a dungeon using Binary Space Partitioning
func (d *Dungeon) GenerateBSP(seed int64, minRoomSize, maxRoomSize int) {
	r := rand.New(rand.NewSource(seed))

	// Create root node covering entire dungeon
	type bspNode struct {
		x, y, w, h  int
		left, right *bspNode
		room        *Room
	}

	var split func(node *bspNode, depth int)
	split = func(node *bspNode, depth int) {
		if depth >= 5 || node.w < minRoomSize*2 || node.h < minRoomSize*2 {
			// Create room in leaf
			roomW := minRoomSize + r.Intn(min(node.w-minRoomSize, maxRoomSize-minRoomSize+1))
			roomH := minRoomSize + r.Intn(min(node.h-minRoomSize, maxRoomSize-minRoomSize+1))
			roomX := node.x + r.Intn(node.w-roomW)
			roomY := node.y + r.Intn(node.h-roomH)

			room := Room{X: roomX, Y: roomY, W: roomW, H: roomH}
			node.room = &room
			d.Rooms = append(d.Rooms, room)

			// Carve room
			for y := roomY; y < roomY+roomH; y++ {
				for x := roomX; x < roomX+roomW; x++ {
					d.Set(x, y, TileFloor)
				}
			}
			return
		}

		// Split horizontally or vertically
		if r.Float32() > 0.5 {
			// Horizontal split
			splitY := node.y + minRoomSize + r.Intn(node.h-minRoomSize*2)
			node.left = &bspNode{x: node.x, y: node.y, w: node.w, h: splitY - node.y}
			node.right = &bspNode{x: node.x, y: splitY, w: node.w, h: node.y + node.h - splitY}
		} else {
			// Vertical split
			splitX := node.x + minRoomSize + r.Intn(node.w-minRoomSize*2)
			node.left = &bspNode{x: node.x, y: node.y, w: splitX - node.x, h: node.h}
			node.right = &bspNode{x: splitX, y: node.y, w: node.x + node.w - splitX, h: node.h}
		}

		split(node.left, depth+1)
		split(node.right, depth+1)
	}

	root := &bspNode{x: 1, y: 1, w: d.Width - 2, h: d.Height - 2}
	split(root, 0)

	// Connect rooms
	for i := 0; i < len(d.Rooms)-1; i++ {
		d.connectRooms(&d.Rooms[i], &d.Rooms[i+1], r)
	}

	// Add stairs
	if len(d.Rooms) >= 2 {
		startRoom := d.Rooms[0]
		endRoom := d.Rooms[len(d.Rooms)-1]
		d.Set(startRoom.X+startRoom.W/2, startRoom.Y+startRoom.H/2, TileStairsUp)
		d.Set(endRoom.X+endRoom.W/2, endRoom.Y+endRoom.H/2, TileStairsDown)
	}
}

func (d *Dungeon) connectRooms(r1, r2 *Room, rng *rand.Rand) {
	// Get center points
	x1 := r1.X + r1.W/2
	y1 := r1.Y + r1.H/2
	x2 := r2.X + r2.W/2
	y2 := r2.Y + r2.H/2

	// Carve L-shaped corridor
	if rng.Float32() > 0.5 {
		d.carveHorizontal(x1, x2, y1)
		d.carveVertical(y1, y2, x2)
	} else {
		d.carveVertical(y1, y2, x1)
		d.carveHorizontal(x1, x2, y2)
	}
}

func (d *Dungeon) carveHorizontal(x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		d.Set(x, y, TileFloor)
	}
}

func (d *Dungeon) carveVertical(y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		d.Set(x, y, TileFloor)
	}
}

// Island generation for ocean/pirate games

// Island represents a generated island
type Island struct {
	CenterX, CenterY float32
	Radius           float32
	Heightmap        *Heightmap
	Biome            string
}

// WorldMap generates a world map with islands
type WorldMap struct {
	Width   int
	Height  int
	Tiles   []float32 // Height values
	Islands []Island
}

// NewWorldMap creates a new world map
func NewWorldMap(width, height int) *WorldMap {
	return &WorldMap{
		Width:  width,
		Height: height,
		Tiles:  make([]float32, width*height),
	}
}

// GenerateArchipelago generates a world with multiple islands
func (w *WorldMap) GenerateArchipelago(seed int64, numIslands int, oceanLevel float32) {
	noise := NewNoise2D(seed)
	r := rand.New(rand.NewSource(seed))

	// Generate base ocean with some variation
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			nx := float64(x) / float64(w.Width) * 4
			ny := float64(y) / float64(w.Height) * 4
			oceanVariation := float32(noise.FBM(nx, ny, 3, 0.5)) * 0.1
			w.Tiles[y*w.Width+x] = oceanLevel + oceanVariation
		}
	}

	// Generate islands
	for i := 0; i < numIslands; i++ {
		cx := float32(r.Intn(w.Width))
		cy := float32(r.Intn(w.Height))
		radius := float32(20 + r.Intn(50))

		island := Island{
			CenterX: cx,
			CenterY: cy,
			Radius:  radius,
			Biome:   randomBiome(r),
		}
		w.Islands = append(w.Islands, island)

		// Add island height to map
		for y := int(cy - radius); y <= int(cy+radius); y++ {
			for x := int(cx - radius); x <= int(cx+radius); x++ {
				if x < 0 || x >= w.Width || y < 0 || y >= w.Height {
					continue
				}

				dx := float64(float32(x) - cx)
				dy := float64(float32(y) - cy)
				dist := math.Sqrt(dx*dx + dy*dy)

				if dist < float64(radius) {
					// Height based on distance from center
					t := 1 - dist/float64(radius)
					// Add noise variation
					nx := float64(x)/float64(w.Width)*8 + float64(i)*100
					ny := float64(y)/float64(w.Height)*8 + float64(i)*100
					noiseVal := noise.FBM(nx, ny, 4, 0.5)

					height := float32(t*t*(3-2*t)) * (0.5 + float32(noiseVal)*0.3)
					idx := y*w.Width + x
					if height > w.Tiles[idx]-oceanLevel {
						w.Tiles[idx] = oceanLevel + height
					}
				}
			}
		}
	}
}

func randomBiome(r *rand.Rand) string {
	biomes := []string{"tropical", "desert", "forest", "swamp", "volcanic", "ice"}
	return biomes[r.Intn(len(biomes))]
}

// Get returns height at (x, y)
func (w *WorldMap) Get(x, y int) float32 {
	if x < 0 || x >= w.Width || y < 0 || y >= w.Height {
		return 0
	}
	return w.Tiles[y*w.Width+x]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
