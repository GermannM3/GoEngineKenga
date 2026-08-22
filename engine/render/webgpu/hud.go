//go:build webgpu && !js

package webgpu

import (
	_ "embed"
	"encoding/binary"
	"image"
	"math"

	"github.com/cogentcore/webgpu/wgpu"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"goenginekenga/engine/render"
)

//go:embed hud.wgsl
var hudWGSL string

// HUD-оверлей WebGPU: квады из render.BuildHUD (полосы + текст) поверх кадра.
// Текст — атлас basicfont.Face7x13 (ASCII), полосы — белая 1×1 текстура.

type hudState struct {
	pipeline *wgpu.RenderPipeline
	uniform  *wgpu.Buffer // screen size (16 байт)
	vbuf     *wgpu.Buffer // квады, растёт по необходимости
	vcap     int          // ёмкость vbuf в байтах
	sampler  *wgpu.Sampler
	atlasTex *wgpu.Texture
	atlasBG  *wgpu.BindGroup
	whiteTex *wgpu.Texture
	whiteBG  *wgpu.BindGroup
}

// buildFontAtlasRGBA растрирует ASCII 32..126 в атлас (белые глифы на прозрачном).
func buildFontAtlasRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, render.HUDAtlasW, render.HUDAtlasH))
	d := &font.Drawer{Dst: img, Src: image.White, Face: basicfont.Face7x13}
	for i := 0; i < render.HUDCharCount; i++ {
		d.Dot = fixed.P(i*render.HUDCellW, render.HUDGlyphH-2) // базовая линия глифа
		d.DrawBytes([]byte{byte(render.HUDFirstChar + i)})
	}
	return img
}

// ensureHUD лениво создаёт pipeline, атлас и буферы HUD. format — формат surface.
func (sc *sceneState) ensureHUD(format wgpu.TextureFormat) error {
	if sc.hud != nil {
		return nil
	}
	fail := func(err error) error {
		return err
	}
	shader, err := sc.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "hud shader",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: hudWGSL},
	})
	if err != nil {
		return fail(err)
	}
	defer shader.Release()

	// Формат вершины: pos f32x2, uv f32x2, color f32x4 (32 байта)
	const stride = 32
	blend := wgpu.BlendState{
		Color: wgpu.BlendComponent{Operation: wgpu.BlendOperationAdd, SrcFactor: wgpu.BlendFactorSrcAlpha, DstFactor: wgpu.BlendFactorOneMinusSrcAlpha},
		Alpha: wgpu.BlendComponent{Operation: wgpu.BlendOperationAdd, SrcFactor: wgpu.BlendFactorOne, DstFactor: wgpu.BlendFactorOneMinusSrcAlpha},
	}
	pl, err := sc.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "HUD Pipeline",
		Vertex: wgpu.VertexState{
			Module: shader, EntryPoint: "vs_main",
			Buffers: []wgpu.VertexBufferLayout{{
				ArrayStride: stride,
				Attributes: []wgpu.VertexAttribute{
					{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
					{Format: wgpu.VertexFormatFloat32x2, Offset: 8, ShaderLocation: 1},
					{Format: wgpu.VertexFormatFloat32x4, Offset: 16, ShaderLocation: 2},
				},
			}},
		},
		Primitive:   wgpu.PrimitiveState{Topology: wgpu.PrimitiveTopologyTriangleList},
		Multisample: wgpu.MultisampleState{Count: 1, Mask: 0xFFFFFFFF},
		Fragment: &wgpu.FragmentState{
			Module: shader, EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{Format: format, Blend: &blend, WriteMask: wgpu.ColorWriteMaskAll},
			},
		},
	})
	if err != nil {
		return fail(err)
	}

	uniform, err := sc.device.CreateBuffer(&wgpu.BufferDescriptor{Label: "hud uniforms", Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst, Size: 16})
	if err != nil {
		pl.Release()
		return fail(err)
	}
	const initialCap = 256 * 1024
	vbuf, err := sc.device.CreateBuffer(&wgpu.BufferDescriptor{Label: "hud verts", Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst, Size: initialCap})
	if err != nil {
		uniform.Release()
		pl.Release()
		return fail(err)
	}

	// Атлас шрифта и белая 1×1 для цветных полос
	atlas := buildFontAtlasRGBA()
	atlasTex, err := sc.createTextureRGBA("hud font atlas", atlas.Rect.Dx(), atlas.Rect.Dy(), atlas.Pix, false)
	if err != nil {
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}
	whiteTex, err := sc.createTextureRGBA("hud white", 1, 1, []byte{255, 255, 255, 255}, false)
	if err != nil {
		atlasTex.Release()
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}

	smp, err := sc.device.CreateSampler(&wgpu.SamplerDescriptor{
		Label:         "hud sampler",
		AddressModeU:  wgpu.AddressModeClampToEdge,
		AddressModeV:  wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MaxAnisotropy: 1,
	})
	if err != nil {
		whiteTex.Release()
		atlasTex.Release()
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}

	bgl := pl.GetBindGroupLayout(0)
	mkBG := func(view *wgpu.TextureView) (*wgpu.BindGroup, error) {
		return sc.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
			Layout: bgl,
			Entries: []wgpu.BindGroupEntry{
				{Binding: 0, Buffer: uniform, Size: 16},
				{Binding: 1, Sampler: smp},
				{Binding: 2, TextureView: view},
			},
		})
	}
	atlasView, err := atlasTex.CreateView(nil)
	if err != nil {
		smp.Release()
		whiteTex.Release()
		atlasTex.Release()
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}
	defer atlasView.Release()
	atlasBG, err := mkBG(atlasView)
	if err != nil {
		smp.Release()
		whiteTex.Release()
		atlasTex.Release()
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}
	whiteView, err := whiteTex.CreateView(nil)
	if err != nil {
		atlasBG.Release()
		smp.Release()
		whiteTex.Release()
		atlasTex.Release()
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}
	defer whiteView.Release()
	whiteBG, err := mkBG(whiteView)
	if err != nil {
		atlasBG.Release()
		smp.Release()
		whiteTex.Release()
		atlasTex.Release()
		vbuf.Release()
		uniform.Release()
		pl.Release()
		return fail(err)
	}

	sc.hud = &hudState{
		pipeline: pl, uniform: uniform, vbuf: vbuf, vcap: initialCap,
		sampler: smp, atlasTex: atlasTex, atlasBG: atlasBG, whiteTex: whiteTex, whiteBG: whiteBG,
	}
	return nil
}

// renderHUD рисует оверлей в surface (после composite, LoadOp:Load).
func (sc *sceneState) renderHUD(encoder *wgpu.CommandEncoder, view *wgpu.TextureView, o *render.HUDOverlay, width, height int, format wgpu.TextureFormat) error {
	if err := sc.ensureHUD(format); err != nil {
		return err
	}
	bars, text := render.BuildHUD(o, width, height)
	total := make([]float32, 0, len(bars)+len(text))
	total = append(total, bars...)
	total = append(total, text...)
	if len(total) == 0 {
		return nil
	}

	hud := sc.hud
	data := make([]byte, len(total)*4)
	for i, f := range total {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(f))
	}
	if len(data) > hud.vcap {
		cap := len(data) * 2
		nb, err := sc.device.CreateBuffer(&wgpu.BufferDescriptor{Label: "hud verts", Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst, Size: uint64(cap)})
		if err != nil {
			return err
		}
		hud.vbuf.Release()
		hud.vbuf = nb
		hud.vcap = cap
	}
	sc.queue.WriteBuffer(hud.vbuf, 0, data)

	ub := make([]byte, 16)
	binary.LittleEndian.PutUint32(ub[0:], math.Float32bits(float32(width)))
	binary.LittleEndian.PutUint32(ub[4:], math.Float32bits(float32(height)))
	sc.queue.WriteBuffer(hud.uniform, 0, ub)

	pass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{View: view, LoadOp: wgpu.LoadOpLoad, StoreOp: wgpu.StoreOpStore},
		},
	})
	pass.SetPipeline(hud.pipeline)
	pass.SetVertexBuffer(0, hud.vbuf, 0, wgpu.WholeSize)

	barsVerts := uint32(len(bars) / 8)
	textVerts := uint32(len(text) / 8)
	if barsVerts > 0 {
		pass.SetBindGroup(0, hud.whiteBG, nil)
		pass.Draw(barsVerts, 1, 0, 0)
	}
	if textVerts > 0 {
		pass.SetBindGroup(0, hud.atlasBG, nil)
		pass.Draw(textVerts, 1, barsVerts, 0)
	}
	pass.End()
	pass.Release()
	return nil
}

// destroyHUD освобождает ресурсы оверлея.
func (sc *sceneState) destroyHUD() {
	if sc.hud == nil {
		return
	}
	h := sc.hud
	h.atlasBG.Release()
	h.whiteBG.Release()
	h.sampler.Release()
	h.atlasTex.Release()
	h.whiteTex.Release()
	h.vbuf.Release()
	h.uniform.Release()
	h.pipeline.Release()
	sc.hud = nil
}
