package animation

import (
	"time"

	"goenginekenga/engine/ecs"
)

// AnimationSystem обновляет анимации спрайтов
type AnimationSystem struct {
	lastUpdateTime time.Time
}

// NewAnimationSystem создает новую систему анимаций
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{
		lastUpdateTime: time.Now(),
	}
}

// Update обновляет все анимации в мире
func (as *AnimationSystem) Update(world *ecs.World) {
	now := time.Now()
	dt := now.Sub(as.lastUpdateTime).Seconds()
	as.lastUpdateTime = now

	entities := world.Entities()
	for _, id := range entities {
		controller, hasController := world.GetAnimationController(id)
		state, hasState := world.GetAnimationState(id)

		if !hasController {
			continue
		}

		// Если у сущности нет состояния анимации, создаем его
		if !hasState {
			initialState := ecs.AnimationState{
				CurrentClip:  controller.DefaultClip,
				CurrentFrame: 0,
				ElapsedTime:  0,
				IsPlaying:    controller.AutoPlay,
				Loop:         true,
				Speed:        1.0,
			}
			world.SetAnimationState(id, initialState)
			continue // Переходим к следующей сущности, чтобы избежать повторной обработки
		}

		// Если анимация не воспроизводится, пропускаем
		if !state.IsPlaying {
			continue
		}

		// Обновляем состояние анимации
		state.ElapsedTime += float32(dt) * state.Speed * controller.PlaybackSpeed

		// Найти текущий клип
		var clip *ecs.AnimationClip
		for _, c := range controller.Clips {
			if c.Name == state.CurrentClip {
				clip = &c
				break
			}
		}

		if clip == nil {
			continue
		}

		// Рассчитать количество кадров
		totalFrames := len(clip.Frames)
		if totalFrames == 0 {
			continue
		}

		// Определить текущий кадр на основе времени
		var frameDuration float32
		if clip.FrameDuration > 0 {
			frameDuration = clip.FrameDuration
		} else if clip.Duration > 0 {
			frameDuration = clip.Duration / float32(totalFrames)
		} else {
			frameDuration = 0.1 // по умолчанию 100мс на кадр
		}

		// Рассчитать номер кадра
		frameIndex := int(state.ElapsedTime/frameDuration) % totalFrames

		// Если анимация не зациклена и мы достигли последнего кадра
		if !clip.Loop && frameIndex == totalFrames-1 {
			if state.ElapsedTime >= clip.Duration {
				state.IsPlaying = false
				frameIndex = totalFrames - 1
			}
		}

		// Обновить состояние
		state.CurrentFrame = frameIndex
		world.SetAnimationState(id, state)

		// Обновить спрайт-рендерер с новым кадром
		sprite, hasSprite := world.GetSpriteRenderer(id)
		if hasSprite && len(clip.Frames) > 0 {
			// Получаем индекс кадра в анимации
			frameIdx := clip.Frames[frameIndex]

			// Определяем размеры кадра
			frameWidth := clip.FrameWidth
			frameHeight := clip.FrameHeight

			// Если размеры не заданы в анимации, используем параметры спрайта
			if frameWidth == 0 || frameHeight == 0 {
				frameWidth = sprite.SrcW
				frameHeight = sprite.SrcH
			}

			// Если и параметры спрайта не заданы, используем размеры по умолчанию
			if frameWidth == 0 || frameHeight == 0 {
				frameWidth = 128
				frameHeight = 128
			}

			// Определяем позицию кадра в спрайт-листе
			srcX := clip.StartX
			srcY := clip.StartY

			// Используем layout для определения позиции кадра
			if clip.Layout == "vertical" {
				// Кадры организованы вертикально
				srcX = clip.StartX + frameIdx*clip.StepX
				srcY = clip.StartY + frameIdx*clip.StepY
			} else if clip.Layout == "grid" {
				// Кадры организованы в сетке
				cols := 10 // используем значение по умолчанию, если не задано иначе
				if clip.StepX > 0 {
					cols = clip.StepX
				}
				srcX = clip.StartX + (frameIdx%cols)*clip.StepX
				srcY = clip.StartY + (frameIdx/cols)*clip.StepY
			} else {
				// По умолчанию - горизонтальный спрайт-лист
				srcX = clip.StartX + frameIdx*clip.StepX
				srcY = clip.StartY
			}

			// Если шаги не заданы, используем размеры кадра
			if clip.StepX == 0 && clip.Layout != "grid" {
				srcX = clip.StartX + frameIdx*frameWidth
			}
			if clip.StepY == 0 && clip.Layout == "vertical" {
				srcY = clip.StartY + frameIdx*frameHeight
			}

			// Обновляем параметры спрайта
			sprite.SrcX = srcX
			sprite.SrcY = srcY
			sprite.SrcW = frameWidth
			sprite.SrcH = frameHeight

			world.SetSpriteRenderer(id, sprite)
		}
	}
}
