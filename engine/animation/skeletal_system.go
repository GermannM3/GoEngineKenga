package animation

import (
	"path/filepath"
	"strings"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/ecs"
)

// SkeletalAnimationSystem обновляет 3D skeletal-анимации: загружает скелет и клипы
// из ассетов (glTF → .skeleton.json / .clip.json), проигрывает текущий клип через
// animation.Animator и пишет финальные bone matrices в ecs.Animator.BoneMatrices —
// их забирает WebGPU-рендерер для аплоада в шейдер.
type SkeletalAnimationSystem struct {
	projectDir string
	states     map[ecs.EntityID]*skeletalState
}

// skeletalState — кэш рантайм-данных аниматора для одной сущности.
type skeletalState struct {
	animator    *Animator
	clips       map[string]*Clip
	signature   string // SkeletonPath|ClipPaths — для инвалидации при hot-reload
	currentClip string
	started     bool
}

// NewSkeletalAnimationSystem создаёт систему. projectDir нужен для резолва путей
// к .skeleton.json и .clip.json (относительно проекта).
func NewSkeletalAnimationSystem(projectDir string) *SkeletalAnimationSystem {
	return &SkeletalAnimationSystem{
		projectDir: projectDir,
		states:     make(map[ecs.EntityID]*skeletalState),
	}
}

// Update обновляет анимации всех сущностей с компонентом Animator.
func (s *SkeletalAnimationSystem) Update(world *ecs.World, dt float32) {
	if world == nil {
		return
	}
	for _, id := range world.Entities() {
		anim, ok := world.GetAnimator(id)
		if !ok || anim.SkeletonPath == "" {
			continue
		}
		st := s.stateFor(id, &anim)
		if st == nil {
			continue
		}
		if st.animator == nil {
			continue
		}

		// Скорость из компонента (1.0 по умолчанию)
		speed := anim.Speed
		if speed <= 0 {
			speed = 1
		}
		if st.animator.CurrentClip != nil {
			st.animator.CurrentClip.Speed = speed
		}

		// Автостарт: первый клип, если не задан
		if !st.started {
			st.started = true
			if anim.Clip == "" && len(anim.ClipPaths) > 0 {
				if first, ok := st.clips[baseName(anim.ClipPaths[0])]; ok {
					st.animator.Play(first)
					st.currentClip = first.Name
				}
			} else if c, ok := st.clips[anim.Clip]; ok {
				st.animator.Play(c)
				st.currentClip = anim.Clip
			}
		}

		// Переключение клипа с кроссфейдом
		if anim.Clip != st.currentClip {
			if c, ok := st.clips[anim.Clip]; ok {
				st.animator.PlayCrossfade(c, 0.15)
				st.currentClip = anim.Clip
			}
		}

		st.animator.Update(dt)

		// Пишем bone matrices (column-major, как в WGSL mat4x4) в компонент
		boneCount := 0
		if st.animator.Skeleton != nil {
			boneCount = ClampBoneCount(len(st.animator.Skeleton.Bones))
		}
		matrices := make([]float32, 0, boneCount*16)
		for i := 0; i < boneCount; i++ {
			m := st.animator.GetBoneMatrix(i)
			matrices = append(matrices, m[:]...)
		}
		anim.BoneMatrices = matrices
		world.SetAnimator(id, anim)
	}
}

// stateFor возвращает (и при необходимости создаёт) рантайм-состояние для сущности.
// Пересоздаёт состояние, если изменились пути к скелету/клипам (hot-reload).
func (s *SkeletalAnimationSystem) stateFor(id ecs.EntityID, anim *ecs.Animator) *skeletalState {
	sig := anim.SkeletonPath + "|" + strings.Join(anim.ClipPaths, ",")
	st, ok := s.states[id]
	if ok && st.signature == sig {
		return st
	}
	st = &skeletalState{
		clips:     make(map[string]*Clip),
		signature: sig,
	}

	skeleton, err := asset.LoadSkeleton(filepath.Join(s.projectDir, anim.SkeletonPath))
	if err != nil {
		s.states[id] = st // кэшируем неудачу: не перезагружаем каждый кадр
		return st
	}
	st.animator = NewAnimator(AssetSkeletonToSkeleton(skeleton))
	for _, p := range anim.ClipPaths {
		clip, err := asset.LoadAnimationClip(filepath.Join(s.projectDir, p))
		if err != nil {
			continue
		}
		converted := AssetClipToClip(clip)
		if converted == nil {
			continue
		}
		st.clips[converted.Name] = converted
	}
	s.states[id] = st
	return st
}

// baseName возвращает имя файла без расширения (используется как имя клипа по умолчанию).
func baseName(path string) string {
	name := path
	if i := strings.LastIndexAny(name, "/\\"); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndex(name, "."); i > 0 {
		name = name[:i]
	}
	return name
}
