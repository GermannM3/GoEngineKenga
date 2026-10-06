package gameplay

import (
	"goenginekenga/engine/audio"
	"goenginekenga/engine/ecs"
	"goenginekenga/engine/input"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/physics"
)

// GameStatus — состояние игрового процесса (победа/поражение/игра).
type GameStatus int

const (
	StatusPlaying GameStatus = iota
	StatusVictory
	StatusDefeat
)

// SFXClips — asset ID звуков боя (резолвятся в run.go по путям исходников).
// Пустая строка = звук не используется.
type SFXClips struct {
	Hit     string // удар по врагу
	Hurt    string // игрок получил урон
	Death   string // враг умер
	Attack  string // взмах игрока (промах/попадание)
	Victory string // победа (портал)
}

// GameLogicSystem handles the core gameplay mechanics: enemy AI,
// pickups, health/damage, win/lose and level flow. Движение игрока
// обрабатывает runtime.ApplyPlayerInput (A/D, Space), ближний бой —
// combat.go (Attacker/Health) — здесь ввод на движение НЕ перехватывается.
type GameLogicSystem struct {
	playerID ecs.EntityID
	enemyIDs []ecs.EntityID
	itemIDs  []ecs.EntityID
	npcIDs   []ecs.EntityID

	// Sound — аудиосистема для звуков действий. Клипы берутся из AudioSource
	// компонентов сущностей: player (прыжок/урон), предметы (подбор) — как в Unity.
	Sound *audio.AudioSystem

	// SFX — звуки боя по asset ID (приоритетнее клипов сущностей).
	SFX SFXClips

	// ShardMeshID — asset ID меша осколков эффектов (кристалл); "" — без эффектов.
	ShardMeshID string

	// Игровой статус и прогресс уровня
	status     GameStatus
	health     float32
	maxHealth  float32
	score      int
	kills      int
	level      int
	invuln     float32 // секунд неуязвимости после урона
	prevVelY   float32 // предыдущая вертикальная скорость игрока (детект прыжка)
	collected  map[ecs.EntityID]bool
	enemyFSMs  map[ecs.EntityID]*enemyFSM
	shards     map[ecs.EntityID]float32 // осколки эффектов: id -> оставшаяся жизнь
	lastWorld  *ecs.World               // смена мира (перезапуск/уровень) сбрасывает состояние

	// Игровой цикл уровня: собери все сферы → открой портал → добеги до него.
	levelTime    float32      // секунд с начала уровня (подсказки управления)
	portalSpawned bool        // все сферы собраны, портал открыт
	PortalPos    emath.Vec3   `json:"-"` // позиция портала (для HUD/тестов)
	portalID     ecs.EntityID

	// Колбэки, устанавливаемые run.go: перезагрузка текущего уровня (R)
	// и переход на следующий (победа, ENTER).
	Restart   func()
	NextLevel func()

	// Хуки для «сока» (тряска камеры, частицы): удар нанесён / игрок ранен.
	OnHit        func(pos emath.Vec3)
	OnPlayerHurt func(pos emath.Vec3)
}

// NewGameLogicSystem creates a new instance of the game logic system
func NewGameLogicSystem() *GameLogicSystem {
	return &GameLogicSystem{
		collected: map[ecs.EntityID]bool{},
		enemyFSMs: map[ecs.EntityID]*enemyFSM{},
		shards:    map[ecs.EntityID]float32{},
	}
}

// SetLevel задаёт номер текущего уровня (1-based) для HUD.
func (gls *GameLogicSystem) SetLevel(n int) {
	gls.level = n
}

// Update processes the game logic for each frame
func (gls *GameLogicSystem) Update(world *ecs.World, inputState *input.State, dt float32) {
	if world == nil {
		return
	}
	gls.findEntities(world)

	// Мир пересоздан (перезапуск уровня, переход на следующий, hot-reload сцены) —
	// entity ID изменились, состояние системы сбрасываем.
	if world != gls.lastWorld {
		gls.lastWorld = world
		gls.collected = map[ecs.EntityID]bool{}
		gls.enemyFSMs = map[ecs.EntityID]*enemyFSM{}
		gls.shards = map[ecs.EntityID]float32{}
		gls.prevVelY = 0
		gls.invuln = 0
		gls.status = StatusPlaying
		gls.score = 0
		gls.kills = 0
		gls.levelTime = 0
		gls.portalSpawned = false
		gls.portalID = 0
		gls.readHealth(world)
	}
	gls.levelTime += dt

	// R — перезапуск уровня в любом состоянии
	if inputState != nil && inputState.IsKeyJustPressed(input.KeyR) {
		gls.restartLevel()
		return
	}

	if gls.status == StatusVictory {
		// ENTER — следующий уровень (на последнем уровне просто остаёмся)
		if inputState != nil && inputState.IsKeyJustPressed(input.KeyEnter) && gls.NextLevel != nil {
			gls.NextLevel()
		}
		return
	}
	if gls.status == StatusDefeat {
		return
	}

	if gls.invuln > 0 {
		gls.invuln -= dt
	}

	tickAttackers(world, dt) // кулдауны атак и i-frames всех бойцов
	gls.updateShards(world, dt)
	gls.updatePlayerAttack(world, inputState)
	gls.updatePlayer(world)
	gls.updateEnemies(world, dt)
	gls.checkItemPickups(world)
	gls.checkHazards(world)
	gls.clampPlayerToArena(world)
	gls.updateGameStatus(world)
}

// clampPlayerToArena — невидимые стены: игрок не покидает арену
// (отброс от врага не должен выбрасывать его в пустоту).
func (gls *GameLogicSystem) clampPlayerToArena(world *ecs.World) {
	if gls.playerID == 0 {
		return
	}
	tr, ok := world.GetTransform(gls.playerID)
	if !ok {
		return
	}
	const bound = 11.0
	clamped := false
	if tr.Position.X > bound {
		tr.Position.X = bound
		clamped = true
	} else if tr.Position.X < -bound {
		tr.Position.X = -bound
		clamped = true
	}
	if tr.Position.Z > bound {
		tr.Position.Z = bound
		clamped = true
	} else if tr.Position.Z < -bound {
		tr.Position.Z = -bound
		clamped = true
	}
	if clamped {
		world.SetTransform(gls.playerID, tr)
		if rb, hasRb := world.GetRigidbody(gls.playerID); hasRb {
			rb.Velocity.X *= -0.3 // мягкий отскок от стены
			rb.Velocity.Z *= -0.3
			world.SetRigidbody(gls.playerID, rb)
		}
	}
}

// updatePlayerAttack: удар игрока по вводу (F / ЛКМ) через Attacker.
func (gls *GameLogicSystem) updatePlayerAttack(world *ecs.World, inputState *input.State) {
	if !playerWantsAttack(inputState) || gls.playerID == 0 {
		return
	}
	atk, ok := world.GetAttacker(gls.playerID)
	if !ok {
		atk = defaultPlayerAttack() // старые сцены без компонента
	}
	if atk.Timer > 0 {
		return
	}
	tr, hasTr := world.GetTransform(gls.playerID)
	if !hasTr {
		return
	}
	atk.Timer = atk.Cooldown
	world.SetAttacker(gls.playerID, atk)

	for _, target := range attackTargetsOf(world, gls.playerID, tr, atk) {
		gls.hitEnemy(world, target, tr.Position, atk.Damage, atk.Knockback)
	}
}

// hitEnemy наносит урон врагу: звук, осколки, счёт убийств, удаление трупа.
func (gls *GameLogicSystem) hitEnemy(world *ecs.World, target ecs.EntityID, from emath.Vec3, dmg, knockback float32) {
	pos := posOf(world, target)
	hit, died := applyHit(world, target, dmg, knockback, from)
	if !hit {
		return // i-frames или цель уже мертва
	}
	if gls.Sound != nil && !died {
		clip := gls.SFX.Hit
		if clip == "" {
			clip = gls.clipOf(world, target)
		}
		if clip != "" {
			gls.Sound.PlayOneShot(clip, pos, 0.7)
		}
	}
	if gls.OnHit != nil {
		gls.OnHit(pos)
	}
	gls.burst(world, pos, 5)
	if died {
		gls.kills++
		gls.burst(world, pos.Add(emath.Vec3{X: 0, Y: 0.4}), 14)
		if gls.Sound != nil {
			clip := gls.SFX.Death
			if clip == "" {
				clip = gls.SFX.Hit
			}
			if clip != "" {
				gls.Sound.PlayOneShot(clip, pos, 0.9)
			}
		}
		world.RemoveEntity(target)
	}
}

// findEntities locates all relevant entities in the world
func (gls *GameLogicSystem) findEntities(world *ecs.World) {
	gls.playerID = 0
	gls.enemyIDs = []ecs.EntityID{}
	gls.itemIDs = []ecs.EntityID{}
	gls.npcIDs = []ecs.EntityID{}

	for _, id := range world.Entities() {
		name := world.Name(id)
		if name == "Player" {
			gls.playerID = id
		} else if containsSubstring(name, "Enemy") {
			gls.enemyIDs = append(gls.enemyIDs, id)
		} else if containsSubstring(name, "Item") {
			gls.itemIDs = append(gls.itemIDs, id)
		} else if containsSubstring(name, "NPC") {
			gls.npcIDs = append(gls.npcIDs, id)
		}
	}
}

// readHealth читает здоровье игрока из компонента Health в мир (при пересоздании мира).
func (gls *GameLogicSystem) readHealth(world *ecs.World) {
	if gls.playerID == 0 {
		return
	}
	if h, ok := world.GetHealth(gls.playerID); ok {
		gls.health = h.Current
		gls.maxHealth = h.Max
	} else {
		gls.health, gls.maxHealth = 0, 0
	}
}

// updatePlayer: звук прыжка. Само движение игрока применяет
// runtime.ApplyPlayerInput (velocity.Y скачком становится 9.0).
func (gls *GameLogicSystem) updatePlayer(world *ecs.World) {
	rb, ok := world.GetRigidbody(gls.playerID)
	if !ok || gls.Sound == nil {
		return
	}
	// Скачок вертикальной скорости = прыжок (отскок от пола после падения < 6)
	if rb.Velocity.Y > 6 && gls.prevVelY <= 0.5 {
		if clip := gls.clipOf(world, gls.playerID); clip != "" {
			pos := emath.V3(0, 0, 0)
			if tr, hasTr := world.GetTransform(gls.playerID); hasTr {
				pos = tr.Position
			}
			gls.Sound.PlayOneShot(clip, pos, 0.5)
		}
	}
	gls.prevVelY = rb.Velocity.Y
}

// updateEnemies ведёт ИИ врагов: FSM Patrol→Chase→Attack (параметры из
// EnemyBrain), в состоянии attack — урон игроку по кулдауну Attacker'а.
func (gls *GameLogicSystem) updateEnemies(world *ecs.World, dt float32) {
	for _, enemyID := range gls.enemyIDs {
		tr, hasTr := world.GetTransform(enemyID)
		if !hasTr {
			continue
		}
		if _, hasRb := world.GetRigidbody(enemyID); !hasRb {
			continue
		}

		fsm := gls.enemyFSMs[enemyID]
		if fsm == nil {
			fsm = newEnemyFSM(world, enemyID, gls.playerID, gls.brainOf(world, enemyID))
			gls.enemyFSMs[enemyID] = fsm
		}
		fsm.agent.Position = tr.Position
		fsm.sm.Update(dt)

		// В состоянии attack наносим урон по кулдауну Attacker'а
		if fsm.agent.State != "attack" {
			continue
		}
		if gls.invuln > 0 || gls.status != StatusPlaying {
			continue // игрок неуязвим или раунд окончен
		}
		atk, ok := world.GetAttacker(enemyID)
		if !ok {
			atk = defaultEnemyAttack()
		}
		if atk.Timer > 0 {
			continue
		}
		atk.Timer = atk.Cooldown
		world.SetAttacker(enemyID, atk)
		gls.damagePlayer(world, tr.Position, atk.Damage)
	}
}

// brainOf возвращает EnemyBrain врага или разумные дефолты (старые сцены).
func (gls *GameLogicSystem) brainOf(world *ecs.World, id ecs.EntityID) ecs.EnemyBrain {
	if b, ok := world.GetEnemyBrain(id); ok {
		return b
	}
	return ecs.EnemyBrain{PatrolSpan: 6, Speed: 2.5, ChaseSpeed: 3.5, AggroRadius: 8, AttackRadius: 1.8}
}

// defaultPlayerAttack / defaultEnemyAttack — параметры для сцен без компонентов.
func defaultPlayerAttack() ecs.Attacker {
	return ecs.Attacker{Radius: 2.2, ArcDeg: 120, Damage: 34, Cooldown: 0.45, Knockback: 7}
}

func defaultEnemyAttack() ecs.Attacker {
	return ecs.Attacker{Radius: 1.8, ArcDeg: 360, Damage: 15, Cooldown: 1.0, Knockback: 5}
}

// damagePlayer отнимает здоровье, включает неуязвимость, отбрасывает игрока.
func (gls *GameLogicSystem) damagePlayer(world *ecs.World, from emath.Vec3, dmg float32) {
	gls.health -= dmg
	if gls.health < 0 {
		gls.health = 0
	}
	gls.invuln = 1.0

	// Пишем здоровье обратно в компонент
	if h, ok := world.GetHealth(gls.playerID); ok {
		h.Current = gls.health
		world.SetHealth(gls.playerID, h)
	}

	// Отбрасывание от врага
	if rb, ok := world.GetRigidbody(gls.playerID); ok {
		dirX, dirZ := float32(1), float32(0)
		if tr, hasTr := world.GetTransform(gls.playerID); hasTr {
			dx := tr.Position.X - from.X
			dz := tr.Position.Z - from.Z
			if dx != 0 || dz != 0 {
				len := emath.V3(dx, 0, dz).Len()
				dirX, dirZ = dx/len, dz/len
			}
		}
		rb.Velocity.X = dirX * 5
		rb.Velocity.Z = dirZ * 5
		rb.Velocity.Y = 7
		world.SetRigidbody(gls.playerID, rb)
	}

	gls.burst(world, posOf(world, gls.playerID).Add(emath.Vec3{X: 0, Y: 0.5}), 8)

	// Звук урона — из SFX или AudioSource игрока
	if gls.Sound != nil {
		clip := gls.SFX.Hurt
		if clip == "" {
			clip = gls.clipOf(world, gls.playerID)
		}
		if clip != "" {
			gls.Sound.PlayOneShot(clip, from, 0.8)
		}
	}
	if gls.OnPlayerHurt != nil {
		gls.OnPlayerHurt(from)
	}
}

// checkHazards наносит урон, если игрок внутри зоны Hazard* (лава/шипы).
func (gls *GameLogicSystem) checkHazards(world *ecs.World) {
	if gls.playerID == 0 || gls.invuln > 0 || gls.status != StatusPlaying {
		return
	}
	p := posOf(world, gls.playerID)
	for _, id := range world.Entities() {
		if !containsSubstring(world.Name(id), "Hazard") {
			continue
		}
		col, ok := world.GetCollider(id)
		if !ok {
			continue
		}
		box := physics.AABBFromCollider(&col, posOf(world, id))
		if p.X >= box.Min.X-0.4 && p.X <= box.Max.X+0.4 &&
			p.Y >= box.Min.Y-0.6 && p.Y <= box.Max.Y+0.3 &&
			p.Z >= box.Min.Z-0.4 && p.Z <= box.Max.Z+0.4 {
			// Источник урона «из-под пола»: отброс вверх
			gls.damagePlayer(world, p.Add(emath.Vec3{X: 0, Y: -1, Z: 0}), 30)
			return
		}
	}
}

// checkItemPickups handles collecting items (3D: дистанция в метрах)
func (gls *GameLogicSystem) checkItemPickups(world *ecs.World) {
	if gls.playerID == 0 {
		return
	}
	playerTr, hasPlayerTr := world.GetTransform(gls.playerID)
	if !hasPlayerTr {
		return
	}

	for _, itemID := range gls.itemIDs {
		if gls.collected[itemID] {
			continue
		}
		itemTr, hasItemTr := world.GetTransform(itemID)
		if !hasItemTr {
			continue
		}
		dx := playerTr.Position.X - itemTr.Position.X
		dy := playerTr.Position.Y - itemTr.Position.Y
		dz := playerTr.Position.Z - itemTr.Position.Z
		if dx*dx+dy*dy+dz*dz < 1.3*1.3 {
			gls.collected[itemID] = true
			gls.score++
			itemPos := itemTr.Position
			// Убираем предмет с карты (вниз под пол)
			itemTr.Position.Y = -100
			world.SetTransform(itemID, itemTr)
			gls.burst(world, itemPos, 6)
			// Звук подбора — из AudioSource предмета
			if gls.Sound != nil {
				if clip := gls.clipOf(world, itemID); clip != "" {
					gls.Sound.PlayOneShot(clip, itemPos, 0.6)
				}
			}
		}
	}
}

// clipOf возвращает asset ID клипа из AudioSource сущности ("" — нет звука).
func (gls *GameLogicSystem) clipOf(world *ecs.World, id ecs.EntityID) string {
	if id == 0 {
		return ""
	}
	src, ok := world.GetAudioSource(id)
	if !ok {
		return ""
	}
	return src.Clip
}

// updateGameStatus: победа через игровой цикл — собери все сферы →
// откроется портал → добеги до него (как в Unity-платформерах).
func (gls *GameLogicSystem) updateGameStatus(world *ecs.World) {
	if gls.health <= 0 {
		gls.status = StatusDefeat
		return
	}
	if !gls.portalSpawned && len(gls.itemIDs) > 0 && gls.score >= len(gls.itemIDs) {
		gls.spawnPortal(world)
	}
	if gls.portalSpawned {
		p := posOf(world, gls.playerID)
		d := p.Sub(gls.PortalPos).Len()
		if d < 1.8 {
			gls.status = StatusVictory
			if gls.Sound != nil && gls.SFX.Victory != "" {
				gls.Sound.PlayOneShot(gls.SFX.Victory, gls.PortalPos, 1.0)
			}
		}
	}
}

// spawnPortal открывает портал: большой вращающийся кристалл в центре арены.
func (gls *GameLogicSystem) spawnPortal(w *ecs.World) {
	gls.portalSpawned = true
	if gls.ShardMeshID == "" || gls.playerID == 0 {
		// Без меша портала — старое поведение (победа сразу после сбора).
		gls.status = StatusVictory
		return
	}
	id := w.CreateEntity("GoalPortal")
	w.SetTransform(id, ecs.Transform{
		Position: emath.V3(0, 1.6, -4),
		Rotation: emath.V3(0, 45, 0),
		Scale:    emath.Vec3{X: 2.8, Y: 2.8, Z: 2.8},
	})
	w.SetMeshRenderer(id, ecs.MeshRenderer{
		MeshAssetID: gls.ShardMeshID,
		ColorR:      160, ColorG: 255, ColorB: 200, ColorA: 255,
	})
	w.SetScript(id, ecs.Script{Name: "spinner", Params: map[string]string{"speed": "260"}})
	gls.portalID = id
	gls.PortalPos = posOf(w, id)
	if gls.Sound != nil && gls.SFX.Victory != "" {
		gls.Sound.PlayOneShot(gls.SFX.Victory, gls.PortalPos, 0.9)
	}
}

// restartLevel сбрасывает прогресс и перезагружает сцену текущего уровня.
func (gls *GameLogicSystem) restartLevel() {
	gls.status = StatusPlaying
	gls.score = 0
	gls.collected = map[ecs.EntityID]bool{}
	gls.enemyFSMs = map[ecs.EntityID]*enemyFSM{}
	gls.invuln = 0
	if gls.Restart != nil {
		gls.Restart()
	}
}

// Status возвращает текущий статус игры.
func (gls *GameLogicSystem) Status() GameStatus {
	return gls.status
}

// Публичные геттеры для графического HUD (полоса здоровья, счёт, статус).

// HealthState возвращает (текущее здоровье, максимум, счёт предметов, всего предметов, уровень, статус).
func (gls *GameLogicSystem) HealthState() (health, maxHealth float32, score, totalItems, level int, status GameStatus) {
	return gls.health, gls.maxHealth, gls.score, len(gls.itemIDs), gls.level, gls.status
}

// Kills возвращает число убитых врагов.
func (gls *GameLogicSystem) Kills() int { return gls.kills }

// GameKills возвращает убийства активной игровой логики (для HUD).
func GameKills() int {
	if defaultGLS == nil {
		return 0
	}
	return defaultGLS.Kills()
}

// GameHealthState возвращает данные активной игровой логики (для HUD-рендера).
func GameHealthState() (health, maxHealth float32, score, totalItems, level int, status GameStatus) {
	if defaultGLS == nil {
		return 0, 0, 0, 0, 0, StatusPlaying
	}
	return defaultGLS.HealthState()
}

// HUD собирает строку интерфейса уровня (здоровье, сферы, номер уровня,
// либо экран победы/поражения). Локализуется через gameplay.Tr.
func (gls *GameLogicSystem) HUD() string {
	switch gls.status {
	case StatusVictory:
		return Tr("hud.victory") + "\n" + Tr("hud.next")
	case StatusDefeat:
		return Tr("hud.defeat") + "\n" + Tr("hud.restart")
	}
	out := Tr("hud.health", int(gls.health))
	out += "  " + Tr("hud.items", gls.score, len(gls.itemIDs))
	if gls.kills > 0 {
		out += "  " + Tr("hud.kills", gls.kills)
	}
	if gls.level > 0 {
		out += "  " + Tr("hud.level", gls.level)
	}
	return out + "\n" + Tr("hud.restart")
}

// ---- Пакетный доступ для HUD рендера (как SetDefaultLocalization) ----

var defaultGLS *GameLogicSystem

// SetGameLogicSystem регистрирует активную игровую логику для HUD.
func SetGameLogicSystem(gls *GameLogicSystem) {
	defaultGLS = gls
}

// GameHUD возвращает HUD активной игровой логики ("" — нет игровой системы).
// Вызывается рендер-бэкендом для нек-картовых миров.
func GameHUD() string {
	if defaultGLS == nil {
		return ""
	}
	return defaultGLS.HUD()
}

// Helper function to check if a string contains a substring
func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if str[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
