// Point light shadow pass for skinned meshes: skin in vertex shader (bones),
// linear depth via frag_depth (distance / far) for correct cubemap-comparison.
// Uniforms: view_proj + model + light_pos + light_far (144 bytes), bone_matrices (64 x mat4x4).
struct VertexInput {
  @location(0) position: vec3<f32>,
  @location(1) normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
  @location(3) joints: vec4<f32>,
  @location(4) weights: vec4<f32>,
}

struct Uniforms {
  view_proj: mat4x4<f32>,
  model: mat4x4<f32>,
  light_pos: vec3<f32>,
  light_far: f32,
}

struct BoneUniforms {
  bone_matrices: array<mat4x4<f32>, 64>,
}

@group(0) @binding(0)
var<uniform> uniforms: Uniforms;
@group(0) @binding(1)
var<uniform> bones: BoneUniforms;

struct VertexOutput {
  @builtin(position) clip_position: vec4<f32>,
  @location(0) world_pos: vec3<f32>,
}

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
  let ji0 = min(i32(in.joints.x), 63);
  let ji1 = min(i32(in.joints.y), 63);
  let ji2 = min(i32(in.joints.z), 63);
  let ji3 = min(i32(in.joints.w), 63);

  let skin_matrix = bones.bone_matrices[ji0] * in.weights.x
                  + bones.bone_matrices[ji1] * in.weights.y
                  + bones.bone_matrices[ji2] * in.weights.z
                  + bones.bone_matrices[ji3] * in.weights.w;

  let world_pos = uniforms.model * skin_matrix * vec4<f32>(in.position, 1.0);
  var out: VertexOutput;
  out.clip_position = uniforms.view_proj * world_pos;
  out.world_pos = world_pos.xyz;
  return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @builtin(frag_depth) f32 {
  return length(in.world_pos - uniforms.light_pos) / uniforms.light_far;
}