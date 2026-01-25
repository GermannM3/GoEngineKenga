//go:build !webgpu

package webgpu

import (
	"fmt"

	"goenginekenga/engine/render"
)

type Backend struct{}

func New(title string, width, height int) *Backend {
	_ = title
	_ = width
	_ = height
	return &Backend{}
}

func (b *Backend) RunLoop(initial *render.Frame) error {
	_ = initial
	return fmt.Errorf("webgpu backend disabled (build with -tags webgpu)")
}
