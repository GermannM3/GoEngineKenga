// GoEngineKenga post-processing shader: bright extract, separable gaussian blur,
// composite (ACES tonemap + bloom + vignette). Fullscreen triangle без vertex buffer.

struct PostUniforms {
  direction: vec2<f32>, // blur: смещение в UV на 1 тексель
  intensity: f32,       // bloom intensity (composite)
  vignette: f32,        // сила виньетки (composite)
}

@group(0) @binding(0)
var scene_tex: texture_2d<f32>;
@group(0) @binding(1)
var smp: sampler;
@group(0) @binding(2)
var bloom_tex: texture_2d<f32>;
@group(0) @binding(3)
var uniforms: PostUniforms;

struct VSOut {
  @builtin(position) pos: vec4<f32>,
  @location(0) uv: vec2<f32>,
}

@vertex
fn vs_fullscreen(@builtin(vertex_index) vi: u32) -> VSOut {
  var p = vec2<f32>(0.0);
  if vi == 0u {
    p = vec2<f32>(-1.0, -1.0);
  } else if vi == 1u {
    p = vec2<f32>(3.0, -1.0);
  } else {
    p = vec2<f32>(-1.0, 3.0);
  }
  var out: VSOut;
  out.pos = vec4<f32>(p, 0.0, 1.0);
  out.uv = p * 0.5 + 0.5;
  return out;
}

// Яркие пиксели для bloom (цветной, по максимальной компоненте).
@fragment
fn fs_bright(in: VSOut) -> @location(0) vec4<f32> {
  let c = textureSample(scene_tex, smp, in.uv).rgb;
  let luma = max(c.r, max(c.g, c.b));
  let b = max(luma - 0.8, 0.0) * c / max(luma, 0.0001) * 1.5;
  return vec4<f32>(b, 1.0);
}

// 9-tap гауссовский блюр вдоль uniforms.direction.
@fragment
fn fs_blur(in: VSOut) -> @location(0) vec4<f32> {
  var acc = textureSample(scene_tex, smp, in.uv).rgb * 0.2270270270;
  acc += textureSample(scene_tex, smp, in.uv + uniforms.direction * 1.3846153846).rgb * 0.3162162162;
  acc += textureSample(scene_tex, smp, in.uv - uniforms.direction * 1.3846153846).rgb * 0.3162162162;
  acc += textureSample(scene_tex, smp, in.uv + uniforms.direction * 3.2307692308).rgb * 0.0702702703;
  acc += textureSample(scene_tex, smp, in.uv - uniforms.direction * 3.2307692308).rgb * 0.0702702703;
  return vec4<f32>(acc, 1.0);
}

// ACES filmic tonemap (Narkowicz 2015).
fn aces(x: vec3<f32>) -> vec3<f32> {
  let a = 2.51;
  let b = 0.03;
  let c = 2.43;
  let d = 0.59;
  let e = 0.14;
  return clamp((x * (a * x + b)) / (x * (c * x + d) + e), vec3<f32>(0.0), vec3<f32>(1.0));
}

// Финальная композиция: ACES + bloom + виньетка.
@fragment
fn fs_composite(in: VSOut) -> @location(0) vec4<f32> {
  var c = textureSample(scene_tex, smp, in.uv).rgb;
  c = aces(c);
  let bloom = textureSample(bloom_tex, smp, in.uv).rgb;
  c += bloom * uniforms.intensity;
  let d = distance(in.uv, vec2<f32>(0.5));
  let vg = 1.0 - uniforms.vignette * smoothstep(0.35, 0.85, d);
  return vec4<f32>(c * vg, 1.0);
}