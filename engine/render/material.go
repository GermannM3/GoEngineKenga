package render

import (
	emath "goenginekenga/engine/math"
)

// Material определяет свойства поверхности для рендеринга
type Material struct {
	Name string `json:"name"`

	// Base properties
	BaseColor            emath.Vec3 `json:"baseColor"`    // RGB color
	BaseColorTex         string     `json:"baseColorTex"` // Texture asset ID
	Metallic             float32    `json:"metallic"`
	Roughness            float32    `json:"roughness"`
	MetallicRoughnessTex string     `json:"metallicRoughnessTex"`

	// Normal mapping
	NormalTex   string  `json:"normalTex"`
	NormalScale float32 `json:"normalScale"`

	// Emissive properties
	EmissiveColor    emath.Vec3 `json:"emissiveColor"`
	EmissiveTex      string     `json:"emissiveTex"`
	EmissiveStrength float32    `json:"emissiveStrength"`

	// Transparency
	AlphaMode   string  `json:"alphaMode"` // OPAQUE, MASK, BLEND
	AlphaCutoff float32 `json:"alphaCutoff"`
	DoubleSided bool    `json:"doubleSided"`
}

// DefaultMaterial возвращает стандартный материал
func DefaultMaterial() *Material {
	return &Material{
		Name:             "Default",
		BaseColor:        emath.V3(0.8, 0.8, 0.8),
		Metallic:         0.0,
		Roughness:        0.5,
		AlphaMode:        "OPAQUE",
		AlphaCutoff:      0.5,
		NormalScale:      1.0,
		EmissiveStrength: 1.0,
	}
}

// Light определяет источник освещения
type Light struct {
	Type       string     `json:"type"` // directional, point, spot
	Color      emath.Vec3 `json:"color"`
	Intensity  float32    `json:"intensity"`
	Range      float32    `json:"range"`      // для point/spot lights
	InnerAngle float32    `json:"innerAngle"` // для spot lights
	OuterAngle float32    `json:"outerAngle"` // для spot lights
}

// DefaultDirectionalLight возвращает стандартный directional light
func DefaultDirectionalLight() *Light {
	return &Light{
		Type:      "directional",
		Color:     emath.V3(1.0, 1.0, 1.0),
		Intensity: 1.0,
	}
}

// DefaultPointLight возвращает стандартный point light
func DefaultPointLight() *Light {
	return &Light{
		Type:      "point",
		Color:     emath.V3(1.0, 1.0, 1.0),
		Intensity: 1.0,
		Range:     10.0,
	}
}
