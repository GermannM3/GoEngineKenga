package api

import (
	"encoding/json"
	"image/color"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
	"goenginekenga/engine/runtime"
)

// loadModelCommand описывает данные для команды load_model.
type loadModelCommand struct {
	// AssetID — прямой ID ассета из .kenga/assets/index.json (предпочтительно).
	AssetID string `json:"asset_id,omitempty"`
	// Path — исходный путь к glTF/GLB (относительно корня проекта), например "assets/robot.gltf".
	// Если AssetID не указан, Manager попытается найти запись в индексе по этому пути.
	Path string `json:"path,omitempty"`

	// EntityID — логический идентификатор сущности (используем как имя в ECS).
	EntityID string `json:"entity_id,omitempty"`
	// Name — человекочитаемое имя сущности, если EntityID не задан.
	Name string `json:"name,omitempty"`
}

type clearSceneCommand struct {
	// ModeTarget позволяет явно указать, какую world чистить: "play" или "edit".
	// По умолчанию — активный мир.
	ModeTarget string `json:"mode,omitempty"`
}

type setTransformCommand struct {
	EntityID string     `json:"entity_id"`
	Position [3]float32 `json:"pos,omitempty"`
	Rotation [3]float32 `json:"rot_deg,omitempty"`
	Scale    [3]float32 `json:"scale,omitempty"`

	// UsePos/UseRot/UseScale позволяют передавать нули, отличая их от "поля не было".
	UsePos   bool `json:"use_pos,omitempty"`
	UseRot   bool `json:"use_rot,omitempty"`
	UseScale bool `json:"use_scale,omitempty"`
}

type setCameraCommand struct {
	Position [3]float32 `json:"pos"`
	Target   [3]float32 `json:"target"`
	FOV      float32    `json:"fov_deg,omitempty"`
	Near     float32    `json:"near,omitempty"`
	Far      float32    `json:"far,omitempty"`
}

type trajectoryPoint [3]float32

type setTrajectoryCommand struct {
	EntityID string           `json:"entity_id"`
	Points   []trajectoryPoint `json:"points"`

	ColorRGBA [4]uint8 `json:"color_rgba,omitempty"`
	Width     float32  `json:"width,omitempty"`
}

type addTrajectoryPointCommand struct {
	EntityID string          `json:"entity_id"`
	Point    trajectoryPoint `json:"point"`
}

type clearTrajectoryCommand struct {
	EntityID string `json:"entity_id"`
}

type setJointCommand struct {
	EntityID  string     `json:"entity_id,omitempty"`
	JointName string     `json:"joint_name,omitempty"`
	AngleDeg  float32    `json:"angle_deg"`
	Axis      [3]float32 `json:"axis,omitempty"`
}

type dispenserCommand struct {
	EntityID string     `json:"entity_id"`
	FlowRate float32    `json:"flow_rate,omitempty"`
	Radius   float32    `json:"radius,omitempty"`
	Color    [4]uint8   `json:"color_rgba,omitempty"`
	Active   bool       `json:"-"`
}

func (m *Manager) ProcessPending(ctx context.Context) {
	if m == nil {
		return
	}
	cmds := m.queue.Drain()
	if len(cmds) == 0 {
		return
	}

	for _, env := range cmds {
		m.handleCommand(ctx, env)
	}
}

func (m *Manager) handleCommand(_ context.Context, env CommandEnvelope) {
	switch env.Cmd {
	case "load_model":
		m.cmdLoadModel(env)
	case "clear_scene":
		m.cmdClearScene(env)
	case "set_transform":
		m.cmdSetTransform(env)
	case "set_camera":
		m.cmdSetCamera(env)
	case "set_trajectory":
		m.cmdSetTrajectory(env)
	case "add_trajectory_point":
		m.cmdAddTrajectoryPoint(env)
	case "clear_trajectory":
		m.cmdClearTrajectory(env)
	case "set_joint":
		m.cmdSetJoint(env)
	case "start_dispensing":
		m.cmdSetDispenser(env, true)
	case "stop_dispensing":
		m.cmdSetDispenser(env, false)
	default:
		// Неизвестные команды пока игнорируем.
	}
}

func (m *Manager) activeWorld() *ecs.World {
	if m == nil || m.rt == nil {
		return nil
	}
	w, err := m.rt.ActiveWorld()
	if err != nil {
		return nil
	}
	return w
}

func (m *Manager) cmdClearScene(env CommandEnvelope) {
	if m == nil || m.rt == nil {
		return
	}

	var payload clearSceneCommand
	_ = json.Unmarshal(env.Data, &payload)

	switch payload.ModeTarget {
	case "play":
		m.rt.PlayWorld = ecs.NewWorld()
	case "edit":
		m.rt.EditWorld = ecs.NewWorld()
	default:
		// Чистим активный мир.
		switch m.rt.Mode {
		case runtime.ModePlay:
			m.rt.PlayWorld = ecs.NewWorld()
		case runtime.ModeEdit:
			m.rt.EditWorld = ecs.NewWorld()
		}
	}
}

func (m *Manager) cmdLoadModel(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload loadModelCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}

	assetID := payload.AssetID
	if assetID == "" && payload.Path != "" && m.projectDir != "" {
		if idx, err := asset.LoadIndex(m.projectDir); err == nil {
			for _, rec := range idx.Assets {
				if rec.SourcePath == payload.Path {
					assetID = rec.ID
					break
				}
			}
		}
	}
	if assetID == "" {
		return
	}

	name := payload.EntityID
	if name == "" {
		name = payload.Name
	}
	if name == "" {
		name = assetID
	}

	id := m.findOrCreateEntityByName(w, name)

	tr, ok := w.GetTransform(id)
	if !ok {
		tr = ecs.Transform{
			Position: emath.V3(0, 0, 0),
			Rotation: emath.V3(0, 0, 0),
			Scale:    emath.V3(1, 1, 1),
		}
	}
	w.SetTransform(id, tr)

	mr := ecs.MeshRenderer{
		MeshAssetID: assetID,
		// Остальное оставляем по умолчанию — цвет задаётся в шейдере/материале.
	}
	w.SetMeshRenderer(id, mr)
}

func (m *Manager) cmdSetTransform(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload setTransformCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}
	if payload.EntityID == "" {
		return
	}

	id, ok := m.findEntityByName(w, payload.EntityID)
	if !ok {
		// Если сущность не найдена — создаём новую.
		id = w.CreateEntity(payload.EntityID)
	}

	tr, has := w.GetTransform(id)
	if !has {
		tr = ecs.Transform{
			Position: emath.V3(0, 0, 0),
			Rotation: emath.V3(0, 0, 0),
			Scale:    emath.V3(1, 1, 1),
		}
	}

	if payload.UsePos {
		tr.Position = emath.V3(payload.Position[0], payload.Position[1], payload.Position[2])
	}
	if payload.UseRot {
		tr.Rotation = emath.V3(payload.Rotation[0], payload.Rotation[1], payload.Rotation[2])
	}
	if payload.UseScale {
		tr.Scale = emath.V3(payload.Scale[0], payload.Scale[1], payload.Scale[2])
	}

	w.SetTransform(id, tr)
}

func (m *Manager) cmdSetCamera(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload setCameraCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}

	// Ищем существующую камеру.
	var camID ecs.EntityID
	var found bool
	for _, id := range w.Entities() {
		if _, ok := w.GetCamera(id); ok {
			camID = id
			found = true
			break
		}
	}
	if !found {
		camID = w.CreateEntity("RemoteCamera")
	}

	tr, hasTr := w.GetTransform(camID)
	if !hasTr {
		tr = ecs.Transform{
			Position: emath.V3(0, 0, 0),
			Rotation: emath.V3(0, 0, 0),
			Scale:    emath.V3(1, 1, 1),
		}
	}
	tr.Position = emath.V3(payload.Position[0], payload.Position[1], payload.Position[2])
	w.SetTransform(camID, tr)

	cam, hasCam := w.GetCamera(camID)
	if !hasCam {
		cam = ecs.Camera{
			FovYDegrees: 60,
			Near:        0.1,
			Far:         1000,
		}
	}
	if payload.FOV > 0 {
		cam.FovYDegrees = payload.FOV
	}
	if payload.Near > 0 {
		cam.Near = payload.Near
	}
	if payload.Far > 0 {
		cam.Far = payload.Far
	}
	w.SetCamera(camID, cam)
}

func (m *Manager) cmdSetTrajectory(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload setTrajectoryCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}
	if payload.EntityID == "" {
		return
	}

	id, ok := m.findEntityByName(w, payload.EntityID)
	if !ok {
		id = w.CreateEntity(payload.EntityID)
	}

	points := make([]emath.Vec3, 0, len(payload.Points))
	for _, p := range payload.Points {
		points = append(points, emath.V3(p[0], p[1], p[2]))
	}

	col := color.RGBA{R: 255, G: 200, B: 80, A: 255}
	if payload.ColorRGBA != [4]uint8{} {
		col = color.RGBA{
			R: payload.ColorRGBA[0],
			G: payload.ColorRGBA[1],
			B: payload.ColorRGBA[2],
			A: payload.ColorRGBA[3],
		}
	}

	width := payload.Width
	if width <= 0 {
		width = 2
	}

	w.SetTrajectory(id, ecs.Trajectory{
		Points: points,
		Color:  col,
		Width:  width,
	})
}

func (m *Manager) cmdAddTrajectoryPoint(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload addTrajectoryPointCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}
	if payload.EntityID == "" {
		return
	}

	id, ok := m.findEntityByName(w, payload.EntityID)
	if !ok {
		id = w.CreateEntity(payload.EntityID)
	}

	traj, _ := w.GetTrajectory(id)
	traj.Points = append(traj.Points, emath.V3(payload.Point[0], payload.Point[1], payload.Point[2]))
	if traj.Color.A == 0 {
		traj.Color = color.RGBA{R: 255, G: 200, B: 80, A: 255}
	}
	if traj.Width <= 0 {
		traj.Width = 2
	}
	w.SetTrajectory(id, traj)
}

func (m *Manager) cmdClearTrajectory(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload clearTrajectoryCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}
	if payload.EntityID == "" {
		return
	}

	id, ok := m.findEntityByName(w, payload.EntityID)
	if !ok {
		return
	}

	traj, ok := w.GetTrajectory(id)
	if !ok {
		return
	}
	traj.Points = nil
	w.SetTrajectory(id, traj)
}

func (m *Manager) cmdSetJoint(env CommandEnvelope) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload setJointCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}
	if payload.EntityID == "" && payload.JointName == "" {
		return
	}

	// Логическое имя сустава: сначала JointName, потом EntityID.
	logicalName := payload.JointName
	if logicalName == "" {
		logicalName = payload.EntityID
	}

	id, ok := m.findEntityByName(w, logicalName)
	if !ok {
		// В v0 без автоматического создания: если сустава нет — игнорируем.
		return
	}

	tr, hasTr := w.GetTransform(id)
	if !hasTr {
		tr = ecs.Transform{
			Position: emath.V3(0, 0, 0),
			Rotation: emath.V3(0, 0, 0),
			Scale:    emath.V3(1, 1, 1),
		}
	}

	axis := payload.Axis
	jointAxis := emath.V3(axis[0], axis[1], axis[2])
	if jointAxis.X == 0 && jointAxis.Y == 0 && jointAxis.Z == 0 {
		// По умолчанию вращаем вокруг локальной оси Y.
		jointAxis = emath.V3(0, 1, 0)
	}

	// Простейшее соответствие оси компонентам Эйлера.
	if absFloat(jointAxis.X) >= absFloat(jointAxis.Y) && absFloat(jointAxis.X) >= absFloat(jointAxis.Z) {
		tr.Rotation.X = payload.AngleDeg
	} else if absFloat(jointAxis.Y) >= absFloat(jointAxis.Z) {
		tr.Rotation.Y = payload.AngleDeg
	} else {
		tr.Rotation.Z = payload.AngleDeg
	}

	w.SetTransform(id, tr)

	// Обновляем вспомогательный компонент Joint для дальнейшей эволюции системы.
	w.SetJoint(id, ecs.Joint{
		Name:  logicalName,
		Axis:  jointAxis,
		Angle: payload.AngleDeg,
	})
}

func absFloat(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (m *Manager) cmdSetDispenser(env CommandEnvelope, active bool) {
	w := m.activeWorld()
	if w == nil {
		return
	}

	var payload dispenserCommand
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return
	}
	if payload.EntityID == "" {
		return
	}

	id, ok := m.findEntityByName(w, payload.EntityID)
	if !ok {
		id = w.CreateEntity(payload.EntityID)
	}

	disp, _ := w.GetDispenser(id)
	disp.Active = active

	if payload.FlowRate > 0 {
		disp.FlowRate = payload.FlowRate
	}
	if payload.Radius > 0 {
		disp.Radius = payload.Radius
	}
	if payload.Color != [4]uint8{} {
		disp.Color = color.RGBA{
			R: payload.Color[0],
			G: payload.Color[1],
			B: payload.Color[2],
			A: payload.Color[3],
		}
	}

	w.SetDispenser(id, disp)
}

func (m *Manager) findOrCreateEntityByName(w *ecs.World, name string) ecs.EntityID {
	if name == "" {
		return w.CreateEntity("")
	}
	if id, ok := m.findEntityByName(w, name); ok {
		return id
	}
	return w.CreateEntity(name)
}

func (m *Manager) findEntityByName(w *ecs.World, name string) (ecs.EntityID, bool) {
	for _, id := range w.Entities() {
		if w.Name(id) == name {
			return id, true
		}
	}
	return 0, false
}

