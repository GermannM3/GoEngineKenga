package asset

// AnimationClip — данные анимационного клипа (из glTF Animation).
type AnimationClip struct {
	Name     string           `json:"name"`
	Duration float32          `json:"duration"`
	Tracks   []AnimationTrack `json:"tracks"`
	Loop     bool             `json:"loop"`
}

// AnimationTrack — трек анимации (узёл/кость + ключевые кадры).
type AnimationTrack struct {
	NodeName  string      `json:"nodeName"`
	Keyframes []Keyframe  `json:"keyframes"`
}

// Keyframe — ключевой кадр анимации.
type Keyframe struct {
	Time     float32  `json:"time"`
	Position [3]float32 `json:"position"`
	Rotation [4]float32 `json:"rotation"` // quaternion xyzw
	Scale    [3]float32 `json:"scale"`
}
