// HUD-оверлей: квады (полосы, текст) в пиксельных координатах поверх кадра.
// Белая текстура 1×1 даёт цветные полосы; атлас шрифта — текст (берётся альфа).
struct HudUniforms {
	screen: vec2<f32>,
	pad: vec2<f32>,
};

@group(0) @binding(0) var<uniform> u: HudUniforms;
@group(0) @binding(1) var smp: sampler;
@group(0) @binding(2) var tex: texture_2d<f32>;

struct VOut {
	@builtin(position) pos: vec4<f32>,
	@location(0) uv: vec2<f32>,
	@location(1) col: vec4<f32>,
};

@vertex
fn vs_main(@location(0) xy: vec2<f32>, @location(1) uv: vec2<f32>, @location(2) col: vec4<f32>) -> VOut {
	var out: VOut;
	let ndc = vec2<f32>(
		xy.x / u.screen.x * 2.0 - 1.0,
		1.0 - xy.y / u.screen.y * 2.0,
	);
	out.pos = vec4<f32>(ndc, 0.0, 1.0);
	out.uv = uv;
	out.col = col;
	return out;
}

@fragment
fn fs_main(in: VOut) -> @location(0) vec4<f32> {
	let a = textureSample(tex, smp, in.uv).a * in.col.a;
	return vec4<f32>(in.col.rgb, a);
}
