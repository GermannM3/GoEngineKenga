// GoEngineKenga point-light shadow shader.
// Пишет линейную глубину (distance / far) через @builtin(frag_depth),
// чтобы сравнение в main shader было корректным для cubemap/matrix-теней.
struct ShadowUniforms {
  view_proj: mat4x4<f32>,
  model: mat4x4<f32>,
  light_pos: vec3<f32>,
  light_far: f32,
}

struct VertexInput {
  @location(0) position: vec3<f32>,
  @location(1) normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
}

struct VertexOutput {
  @builtin(position) clip_position: vec4<f32>,
  @location(0) world_pos: vec3<f32>,
}

@group(0) @binding(0)
var<uniform> uniforms: ShadowUniforms;

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
  var out: VertexOutput;
  out.clip_position = uniforms.view_proj * uniforms.model * vec4<f32>(in.position, 1.0);
  out.world_pos = (uniforms.model * vec4<f32>(in.position, 1.0)).xyz;
  return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @builtin(frag_depth) f32 {
  return length(in.world_pos - uniforms.light_pos) / uniforms.light_far;
}