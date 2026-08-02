package gameplay

import (
	"math"
	"sync"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
)

const (
	kartFriction    = 45.0 // замедление без газа (px/с²)
	kartDriftFactor = 0.7
	LapsToWin       = 3
	pickupRadius    = 60.0
	waypointRadius  = 100.0
)

// RacePhase состояние гонки
type RacePhase int

const (
	PhaseMenu RacePhase = iota
	PhaseCountdown
	PhaseRacing
	PhaseFinished
)

// RaceState глобальное состояние картинга (для HUD и меню)
type RaceState struct {
	mu           sync.RWMutex
	Phase        RacePhase
	Countdown    float32 // секунды до старта
	SelectedCar  string  // atom | moskvich_m70 | moskvich_m90
	WinnerName   string
	Message      string
	RaceTime     float32 // секунды с момента старта (PhaseRacing)
	goTimer      float32 // таймер показа надписи GO! после старта
	Waypoints    []emath.Vec3
	TrackCenterX float32
	TrackCenterY float32
	TrackRX      float32
	TrackRY      float32
}

var raceState = &RaceState{
	Phase:        PhaseMenu,
	SelectedCar:  "atom",
	TrackCenterX: 640,
	TrackCenterY: 360,
	TrackRX:      420,
	TrackRY:      240,
}

// GetRaceState возвращает состояние гонки
func GetRaceState() *RaceState {
	return raceState
}

// Snapshot читает UI-поля без гонок
func (rs *RaceState) Snapshot() (phase RacePhase, message, selectedCar, winner string) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.Phase, rs.Message, rs.SelectedCar, rs.WinnerName
}

// Elapsed возвращает время гонки (с момента старта), для HUD.
func (rs *RaceState) Elapsed() float32 {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.RaceTime
}

func (rs *RaceState) ensureWaypoints() {
	if len(rs.Waypoints) > 0 {
		return
	}
	// Овал по часовой, старт снизу (угол ~-90° / 270°)
	n := 16
	rs.Waypoints = make([]emath.Vec3, n)
	for i := 0; i < n; i++ {
		// старт внизу овала, движение против часовой → растущий угол от -π/2
		a := -math.Pi/2 + float64(i)*2*math.Pi/float64(n)
		rs.Waypoints[i] = emath.Vec3{
			X: rs.TrackCenterX + rs.TrackRX*float32(math.Cos(a)),
			Y: rs.TrackCenterY + rs.TrackRY*float32(math.Sin(a)),
			Z: 0,
		}
	}
}

// ResetRace готовит новую гонку
func ResetRace(w *ecs.World) {
	rs := raceState
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.ensureWaypoints()
	rs.Phase = PhaseCountdown
	rs.Countdown = 3.5
	rs.WinnerName = ""
	rs.Message = "3"
	rs.RaceTime = 0
	rs.goTimer = 0
	cx, cy := rs.TrackCenterX, rs.TrackCenterY
	_, ry := rs.TrackRX, rs.TrackRY
	wp0 := rs.Waypoints[0]
	wp1 := rs.Waypoints[1]
	heading := float32(math.Atan2(float64(wp1.Y-wp0.Y), float64(wp1.X-wp0.X)) * 180 / math.Pi)

	carIDs := []string{rs.SelectedCar, "moskvich_m70", "moskvich_m90"}
	botIdx := 0
	offsets := []float32{-50, 0, 50}
	for _, id := range w.Entities() {
		kart, ok := w.GetKart(id)
		if !ok {
			continue
		}
		tr, _ := w.GetTransform(id)
		rb, hasRb := w.GetRigidbody(id)
		if !kart.IsBot {
			kart.CarID = rs.SelectedCar
			if sr, ok := w.GetSpriteRenderer(id); ok {
				sr.TexturePath = carTexture(rs.SelectedCar)
				w.SetSpriteRenderer(id, sr)
			}
			off := offsets[1]
			tr.Position.X = cx + off
			tr.Position.Y = cy + ry - 10
		} else {
			botIdx++
			ci := botIdx
			if ci >= len(carIDs) {
				ci = 1
			}
			kart.CarID = carIDs[ci]
			if sr, ok := w.GetSpriteRenderer(id); ok {
				sr.TexturePath = carTexture(kart.CarID)
				w.SetSpriteRenderer(id, sr)
			}
			off := offsets[botIdx%len(offsets)]
			tr.Position.X = cx + off
			tr.Position.Y = cy + ry + float32(botIdx)*25
		}
		tr.Rotation.Z = heading
		kart.CurrentLap = 0
		kart.LastCheckpoint = 0
		kart.PowerUp = ""
		kart.PowerUpTimer = 0
		kart.RacePosition = 1
		kart.LapStartTime = 0
		kart.LastLapTime = 0
		kart.BestLapTime = 0
		if hasRb {
			rb.Velocity = emath.Vec3{}
			rb.UseGravity = false
			rb.Drag = 0
			w.SetRigidbody(id, rb)
		}
		w.SetKart(id, kart)
		w.SetTransform(id, tr)
	}
}

func carTexture(carID string) string {
	switch carID {
	case "moskvich_m70":
		return "assets/cars/moskvich_m70.png"
	case "moskvich_m90":
		return "assets/cars/moskvich_m90.png"
	default:
		return "assets/cars/atom.png"
	}
}

// KartSystem — полный цикл: меню, countdown, гонка, боты по waypoints, бонусы, лапы.
func KartSystem(w *ecs.World, inputState interface{}, dt float32) {
	if w == nil || dt <= 0 {
		return
	}
	st, _ := inputState.(*input.State)
	rs := raceState
	rs.mu.Lock()
	rs.ensureWaypoints()
	rs.mu.Unlock()

	switch rs.Phase {
	case PhaseMenu:
		updateMenu(w, st)
		return
	case PhaseCountdown:
		updateCountdown(w, dt)
		return
	case PhaseFinished:
		if st != nil && (st.IsKeyJustPressed(input.KeyEnter) || st.IsKeyJustPressed(input.KeySpace)) {
			rs.mu.Lock()
			rs.Phase = PhaseMenu
			rs.Message = ""
			rs.mu.Unlock()
		}
		return
	}

	// PhaseRacing
	rs.RaceTime += dt
	// Надпись GO! показываем 1.5с после старта, затем убираем
	if rs.Message != "" {
		rs.goTimer += dt
		if rs.goTimer > 1.5 {
			rs.Message = ""
			rs.goTimer = 0
		}
	}
	updateKarts(w, st, dt, rs)
	updatePickups(w, dt)
	updateLapsAndPositions(w, rs)
	checkFinish(w, rs)
}

func updateMenu(w *ecs.World, st *input.State) {
	rs := raceState
	if st != nil {
		if st.IsKeyJustPressed(input.Key1) {
			rs.mu.Lock()
			rs.SelectedCar = "atom"
			rs.mu.Unlock()
		}
		if st.IsKeyJustPressed(input.Key2) {
			rs.mu.Lock()
			rs.SelectedCar = "moskvich_m70"
			rs.mu.Unlock()
		}
		if st.IsKeyJustPressed(input.Key3) {
			rs.mu.Lock()
			rs.SelectedCar = "moskvich_m90"
			rs.mu.Unlock()
		}
		if st.IsKeyJustPressed(input.KeyEnter) || st.IsKeyJustPressed(input.KeySpace) {
			ResetRace(w)
			return
		}
	}
	rs.mu.Lock()
	rs.Message = Tr("race.menu", rs.SelectedCar)
	rs.mu.Unlock()
}

func updateCountdown(w *ecs.World, dt float32) {
	rs := raceState
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.Countdown -= dt
	n := int(rs.Countdown) + 1
	if n > 3 {
		n = 3
	}
	if rs.Countdown <= 0 {
		rs.Phase = PhaseRacing
		rs.Message = Tr("race.go")
		rs.Countdown = 0
	} else if n <= 0 {
		rs.Message = Tr("race.go")
	} else {
		rs.Message = itoa(n)
	}
	_ = w
}

func updateKarts(w *ecs.World, st *input.State, dt float32, rs *RaceState) {
	waypoints := rs.Waypoints
	for _, id := range w.Entities() {
		kart, ok := w.GetKart(id)
		if !ok {
			continue
		}
		tr, hasTr := w.GetTransform(id)
		if !hasTr {
			continue
		}
		rb, hasRb := w.GetRigidbody(id)
		if !hasRb {
			continue
		}
		cfg, ok := CarConfigByID(kart.CarID)
		if !ok {
			continue
		}

		maxSpeed := cfg.MaxSpeed
		accel := cfg.Acceleration
		// активный nitro / electric_boost
		boosting := kart.PowerUpTimer > 0 && (kart.PowerUp == "nitro" || kart.PowerUp == "electric_boost")
		if boosting {
			maxSpeed *= 1.45
			accel *= 1.6
		}

		angleRad := tr.Rotation.Z * float32(math.Pi) / 180
		forward := emath.Vec3{
			X: float32(math.Cos(float64(angleRad))),
			Y: float32(math.Sin(float64(angleRad))),
		}
		speed := float32(math.Sqrt(float64(rb.Velocity.X*rb.Velocity.X + rb.Velocity.Y*rb.Velocity.Y)))

		if kart.IsBot {
			targetIdx := (kart.LastCheckpoint + 1) % len(waypoints)
			target := waypoints[targetIdx]
			dx := target.X - tr.Position.X
			dy := target.Y - tr.Position.Y
			desired := float32(math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi)
			diff := normalizeAngle(desired - tr.Rotation.Z)
			steer := float32(0)
			if diff > 3 {
				steer = 1
			} else if diff < -3 {
				steer = -1
			}
			tr.Rotation.Z += steer * cfg.Handling * 70 * dt
			angleRad = tr.Rotation.Z * float32(math.Pi) / 180
			forward.X = float32(math.Cos(float64(angleRad)))
			forward.Y = float32(math.Sin(float64(angleRad)))
			speed += accel * 0.75 * dt
			if speed > maxSpeed*0.92 {
				speed = maxSpeed * 0.92
			}
		} else {
			throttle, brake, steer := float32(0), float32(0), float32(0)
			if st != nil {
				if st.IsKeyPressed(input.KeyW) || st.IsKeyPressed(input.KeyArrowUp) {
					throttle = 1
				}
				if st.IsKeyPressed(input.KeyS) || st.IsKeyPressed(input.KeyArrowDown) {
					brake = 1
				}
				if st.IsKeyPressed(input.KeyA) || st.IsKeyPressed(input.KeyArrowLeft) {
					steer = -1
				}
				if st.IsKeyPressed(input.KeyD) || st.IsKeyPressed(input.KeyArrowRight) {
					steer = 1
				}
				// Space — активировать бонус
				if st.IsKeyJustPressed(input.KeySpace) && kart.PowerUp != "" && kart.PowerUpTimer <= 0 {
					activatePowerUp(&kart, &cfg)
				}
			}
			drift := st != nil && (st.IsKeyPressed(input.KeyShiftLeft) || st.IsKeyPressed(input.KeyShiftRight))
			handling := cfg.Handling
			if drift {
				handling *= kartDriftFactor
			}
			speedFactor := float32(0.25)
			if maxSpeed > 0 {
				speedFactor = 0.25 + 0.75*(speed/maxSpeed)
			}
			tr.Rotation.Z += steer * handling * 75 * dt * speedFactor
			angleRad = tr.Rotation.Z * float32(math.Pi) / 180
			forward.X = float32(math.Cos(float64(angleRad)))
			forward.Y = float32(math.Sin(float64(angleRad)))
			speed += accel * (throttle - brake*1.3) * dt
			if speed < 0 {
				speed = 0
			}
			if speed > maxSpeed {
				speed = maxSpeed
			}
			if throttle <= 0 && brake <= 0 {
				speed -= kartFriction * dt
				if speed < 0 {
					speed = 0
				}
			}
		}

		if kart.PowerUpTimer > 0 {
			kart.PowerUpTimer -= dt
			if kart.PowerUpTimer <= 0 {
				kart.PowerUp = ""
				kart.PowerUpTimer = 0
			}
		}

		rb.Velocity.X = forward.X * speed
		rb.Velocity.Y = forward.Y * speed
		rb.Velocity.Z = 0
		rb.UseGravity = false
		rb.Drag = 0

		// Держим на трассе (мягкий pull к овалу)
		pullToOval(&tr, rs, speed, dt)

		w.SetKart(id, kart)
		w.SetTransform(id, tr)
		w.SetRigidbody(id, rb)
	}
}

func activatePowerUp(kart *ecs.Kart, cfg *CarConfig) {
	switch kart.PowerUp {
	case "nitro":
		kart.PowerUpTimer = 2.5
	case "electric_boost":
		if cfg.SpecialAbility == "electric_boost" {
			kart.PowerUpTimer = 3.5
		} else {
			kart.PowerUpTimer = 2.0
			kart.PowerUp = "nitro"
		}
	case "shield":
		kart.PowerUpTimer = 5
	case "oil", "rocket":
		kart.PowerUpTimer = 1
	default:
		kart.PowerUpTimer = 2
	}
}

// pickupDuration — длительность бонуса при авто-активации ботом.
func pickupDuration(t string) float32 {
	switch t {
	case "nitro":
		return 2.5
	case "electric_boost":
		return 3.0
	case "shield":
		return 5
	case "oil", "rocket":
		return 1
	default:
		return 2
	}
}

func pullToOval(tr *ecs.Transform, rs *RaceState, speed, dt float32) {
	dx := tr.Position.X - rs.TrackCenterX
	dy := tr.Position.Y - rs.TrackCenterY
	// нормализованный «радиус» эллипса
	nx := dx / rs.TrackRX
	ny := dy / rs.TrackRY
	r := float32(math.Sqrt(float64(nx*nx + ny*ny)))
	if r < 0.55 {
		// слишком внутрь — выталкиваем
		if r > 0.01 {
			tr.Position.X += dx / r * 40 * dt
			tr.Position.Y += dy / r * 40 * dt
		}
	} else if r > 1.15 {
		// слишком снаружи — тянем внутрь
		tr.Position.X -= dx / r * 80 * dt
		tr.Position.Y -= dy / r * 80 * dt
		// штраф скорости обрабатывается снаружи через friction — просто тормозим через позицию
		_ = speed
	}
}

func updatePickups(w *ecs.World, dt float32) {
	types := []string{"nitro", "shield", "nitro", "electric_boost"}
	ti := 0
	for _, pid := range w.Entities() {
		pickup, ok := w.GetPowerUpPickup(pid)
		if !ok {
			continue
		}
		if pickup.Cooldown > 0 {
			pickup.Cooldown -= dt
			w.SetPowerUpPickup(pid, pickup)
			if pickup.Cooldown <= 0 {
				if sr, ok := w.GetSpriteRenderer(pid); ok {
					sr.Visible = true
					w.SetSpriteRenderer(pid, sr)
				}
			}
			continue
		}
		ptr, ok := w.GetTransform(pid)
		if !ok {
			continue
		}
		for _, kid := range w.Entities() {
			kart, ok := w.GetKart(kid)
			if !ok {
				continue
			}
			if kart.PowerUp != "" {
				continue
			}
			// Боты подбирают бонусы реже и используют сразу (у игрока — по Space)
			if kart.IsBot {
				if (kart.RacePosition % 2) == 0 {
					continue
				}
			}
			tr, _ := w.GetTransform(kid)
			dx := tr.Position.X - ptr.Position.X
			dy := tr.Position.Y - ptr.Position.Y
			if dx*dx+dy*dy < pickupRadius*pickupRadius {
				t := pickup.Type
				if t == "" {
					t = types[ti%len(types)]
				}
				kart.PowerUp = t
				kart.PowerUpTimer = 0 // ждёт Space (у игрока)
				if kart.IsBot {
					// бот активирует бонус немедленно
					kart.PowerUpTimer = pickupDuration(t)
				}
				w.SetKart(kid, kart)
				pickup.Cooldown = pickup.RespawnSec
				if pickup.Cooldown <= 0 {
					pickup.Cooldown = 8
				}
				w.SetPowerUpPickup(pid, pickup)
				if sr, ok := w.GetSpriteRenderer(pid); ok {
					sr.Visible = false
					w.SetSpriteRenderer(pid, sr)
				}
				break
			}
		}
		ti++
	}
}

func updateLapsAndPositions(w *ecs.World, rs *RaceState) {
	waypoints := rs.Waypoints
	n := len(waypoints)
	if n == 0 {
		return
	}
	type prog struct {
		id       ecs.EntityID
		progress float32
	}
	var list []prog
	for _, id := range w.Entities() {
		kart, ok := w.GetKart(id)
		if !ok {
			continue
		}
		tr, _ := w.GetTransform(id)
		next := (kart.LastCheckpoint + 1) % n
		t := waypoints[next]
		dx := tr.Position.X - t.X
		dy := tr.Position.Y - t.Y
		if dx*dx+dy*dy < waypointRadius*waypointRadius {
			prev := kart.LastCheckpoint
			kart.LastCheckpoint = next
			// полный круг: перешли с последнего на 0
			if prev == n-1 && next == 0 {
				kart.CurrentLap++
				// тайминг круга: LapStartTime ставится на старте гонки/предыдущего круга
				if kart.LapStartTime >= 0 {
					lapTime := rs.RaceTime - kart.LapStartTime
					kart.LastLapTime = lapTime
					if kart.BestLapTime <= 0 || lapTime < kart.BestLapTime {
						kart.BestLapTime = lapTime
					}
				}
				kart.LapStartTime = rs.RaceTime
			}
			w.SetKart(id, kart)
		}
		p := float32(kart.CurrentLap)*float32(n) + float32(kart.LastCheckpoint)
		list = append(list, prog{id: id, progress: p})
	}
	// сортировка позиций
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].progress > list[i].progress {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	for i, p := range list {
		kart, _ := w.GetKart(p.id)
		kart.RacePosition = i + 1
		w.SetKart(p.id, kart)
	}
}

func checkFinish(w *ecs.World, rs *RaceState) {
	for _, id := range w.Entities() {
		kart, ok := w.GetKart(id)
		if !ok || kart.IsBot {
			continue
		}
		if kart.CurrentLap >= LapsToWin {
			rs.mu.Lock()
			rs.Phase = PhaseFinished
			rs.WinnerName = w.Name(id)
			if rs.WinnerName == "" {
				rs.WinnerName = "Player"
			}
			rs.Message = Tr("race.finish", itoa(kart.RacePosition))
			rs.mu.Unlock()
			return
		}
	}
	// бот финишировал первым
	for _, id := range w.Entities() {
		kart, ok := w.GetKart(id)
		if !ok || !kart.IsBot {
			continue
		}
		if kart.CurrentLap >= LapsToWin {
			rs.mu.Lock()
			rs.Phase = PhaseFinished
			rs.WinnerName = w.Name(id)
			rs.Message = Tr("race.win", w.Name(id))
			rs.mu.Unlock()
			return
		}
	}
}

func normalizeAngle(a float32) float32 {
	for a > 180 {
		a -= 360
	}
	for a < -180 {
		a += 360
	}
	return a
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [16]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// HasKart возвращает true, если в мире есть хотя бы один Kart
func HasKart(w *ecs.World) bool {
	if w == nil {
		return false
	}
	for _, id := range w.Entities() {
		if _, ok := w.GetKart(id); ok {
			return true
		}
	}
	return false
}

// TrackGeometry для отрисовки овала
func TrackGeometry() (cx, cy, rx, ry float32) {
	rs := raceState
	return rs.TrackCenterX, rs.TrackCenterY, rs.TrackRX, rs.TrackRY
}

// WaypointsForDraw копия waypoints
func WaypointsForDraw() []emath.Vec3 {
	rs := raceState
	rs.ensureWaypoints()
	out := make([]emath.Vec3, len(rs.Waypoints))
	copy(out, rs.Waypoints)
	return out
}
