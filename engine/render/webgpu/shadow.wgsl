// Shadow map pass: render depth from light's view
struct VertexInput {
  @location(0) position: vec3<f32>,
  @location(1) normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
}

struct Uniforms {
  light_view_proj: mat4x4<f32>,
  model: mat4x4<f32>,
}

@group(0) @binding(0)
var<uniform> uniforms: Uniforms;

@vertex
fn vs_main(in: VertexInput) -> @builtin(position) vec4<f32> {
  return uniforms.light_view_proj * uniforms.model * vec4<f32>(in.position, 1.0);
}
