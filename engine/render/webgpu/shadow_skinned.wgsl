// Shadow map pass for skinned meshes: skin in vertex shader (bones), depth only.
// Uniforms: light_view_proj + model (128 bytes), bone_matrices (64 x mat4x4).
struct VertexInput {
  @location(0) position: vec3<f32>,
  @location(1) normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
  @location(3) joints: vec4<f32>,
  @location(4) weights: vec4<f32>,
}

struct Uniforms {
  light_view_proj: mat4x4<f32>,
  model: mat4x4<f32>,
}

struct BoneUniforms {
  bone_matrices: array<mat4x4<f32>, 64>,
}

@group(0) @binding(0)
var<uniform> uniforms: Uniforms;
@group(0) @binding(1)
var<uniform> bones: BoneUniforms;

@vertex
fn vs_main(in: VertexInput) -> @builtin(position) vec4<f32> {
  // Joints хранятся как float bits uint16-индексов; i32() корректен для < 2^24.
  let ji0 = min(i32(in.joints.x), 63);
  let ji1 = min(i32(in.joints.y), 63);
  let ji2 = min(i32(in.joints.z), 63);
  let ji3 = min(i32(in.joints.w), 63);

  let skin_matrix = bones.bone_matrices[ji0] * in.weights.x
                  + bones.bone_matrices[ji1] * in.weights.y
                  + bones.bone_matrices[ji2] * in.weights.z
                  + bones.bone_matrices[ji3] * in.weights.w;

  let world_pos = uniforms.model * skin_matrix * vec4<f32>(in.position, 1.0);
  return uniforms.light_view_proj * world_pos;
}
