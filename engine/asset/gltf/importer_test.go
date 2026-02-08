package gltf

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestImportAnimatedCube(t *testing.T) {
	// AnimatedCube from qmuntal/gltf testdata (in module cache)
	_, self, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(self)
	path := filepath.Join(pkgDir, "..", "..", "..", "samples", "hello", "assets", "AnimatedCube", "AnimatedCube.gltf")
	res, err := ImportFile(path)
	if err != nil {
		t.Skipf("AnimatedCube not found: %v", err)
		return
	}
	if len(res.Meshes) == 0 {
		t.Fatal("expected at least one mesh")
	}
	if len(res.Animations) == 0 {
		t.Fatal("expected at least one animation")
	}
	anim := res.Animations[0]
	if anim.Duration <= 0 {
		t.Errorf("expected positive duration, got %f", anim.Duration)
	}
	if len(anim.Channels) == 0 {
		t.Fatal("expected at least one animation channel")
	}
	ch := anim.Channels[0]
	if ch.Path != "rotation" {
		t.Errorf("expected rotation channel, got %s", ch.Path)
	}
	if len(ch.Times) == 0 || len(ch.Values) == 0 {
		t.Errorf("expected keyframe data, got times=%d values=%d", len(ch.Times), len(ch.Values))
	}
}
