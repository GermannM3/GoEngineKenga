// GoEngineKenga WebGPU PBR shader — skeletal skinning
// Vertex: position, normal, uv, joints (vec4<f32>), weights (vec4<f32>)
// Uniform: viewProj, Material, Light, bone_matrices[64]

struct VertexInput {
  @location(0) position: vec3<f32>,
  @location(1) normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
  @location(3) joints: vec4<f32>,
  @location(4) weights: vec4<f32>,
}

struct VertexOutput {
  @builtin(position) clip_position: vec4<f32>,
  @location(0) world_pos: vec3<f32>,
  @location(1) world_normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
}

struct Uniforms {
  view_proj: mat4x4<f32>,
  model: mat4x4<f32>,
  base_color: vec3<f32>,
  metallic: f32,
  roughness: f32,
  light_dir: vec3<f32>,
  light_intensity: f32,
  light_color: vec3<f32>,
  ambient: f32,
  cam_pos: vec3<f32>,
  _pad: f32,
  light_view_proj: mat4x4<f32>,
}

struct BoneUniforms {
  bone_matrices: array<mat4x4<f32>, 64>,
}

@group(0) @binding(0)
var<uniform> uniforms: Uniforms;
@group(0) @binding(1)
var<uniform> bones: BoneUniforms;

@group(1) @binding(0)
var shadow_map: texture_depth_2d;
@group(1) @binding(1)
var shadow_sampler: sampler_comparison;

fn fresnel_schlick(cos_theta: f32, f0: vec3<f32>) -> vec3<f32> {
  return f0 + (1.0 - f0) * pow(1.0 - cos_theta, 5.0);
}

fn distribution_ggx(n: vec3<f32>, h: vec3<f32>, roughness: f32) -> f32 {
  let a = roughness * roughness * roughness * roughness;
  let a2 = a * a;
  let ndoth = max(dot(n, h), 0.0);
  let ndoth2 = ndoth * ndoth;
  let denom = ndoth2 * (a2 - 1.0) + 1.0;
  return select(0.0, a2 / (3.14159265 * denom * denom), denom > 0.0);
}

fn geometry_schlick_ggx(ndotv: f32, roughness: f32) -> f32 {
  let r = roughness + 1.0;
  let k = r * r / 8.0;
  return ndotv / (ndotv * (1.0 - k) + k);
}

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
  let ji0 = i32(in.joints.x);
  let ji1 = i32(in.joints.y);
  let ji2 = i32(in.joints.z);
  let ji3 = i32(in.joints.w);

  let w0 = in.weights.x;
  let w1 = in.weights.y;
  let w2 = in.weights.z;
  let w3 = in.weights.w;

  let b0 = bones.bone_matrices[ji0];
  let b1 = bones.bone_matrices[ji1];
  let b2 = bones.bone_matrices[ji2];
  let b3 = bones.bone_matrices[ji3];

  let skin_matrix = b0 * w0 + b1 * w1 + b2 * w2 + b3 * w3;
  let skin_pos = skin_matrix * vec4<f32>(in.position, 1.0);
  let skin_normal = skin_matrix * vec4<f32>(in.normal, 0.0);

  let world_pos = uniforms.model * skin_pos;
  let mvp = uniforms.view_proj * world_pos;
  var out: VertexOutput;
  out.clip_position = mvp;
  out.world_pos = world_pos.xyz;
  out.world_normal = (uniforms.model * skin_normal).xyz;
  out.uv = in.uv;
  return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
  let n = normalize(in.world_normal);
  let v = normalize(uniforms.cam_pos - in.world_pos);
  let l = normalize(uniforms.light_dir);
  let h = normalize(v + l);

  let ndotv = max(dot(n, v), 0.0001);
  let ndotl = max(dot(n, l), 0.0);

  let f0 = mix(vec3<f32>(0.04), uniforms.base_color, uniforms.metallic);
  let diffuse = uniforms.base_color * (1.0 - uniforms.metallic);
  let diffuse_term = diffuse * ndotl;
  let d = distribution_ggx(n, h, max(uniforms.roughness, 0.04));
  let f = fresnel_schlick(max(dot(h, v), 0.0), f0);
  let kd = (1.0 - f) * (1.0 - uniforms.metallic);
  let specular = f * d * 0.25;

  let light_clip = uniforms.light_view_proj * vec4<f32>(in.world_pos, 1.0);
  let shadow_z = light_clip.z / light_clip.w;
  let shadow_uv = light_clip.xy / light_clip.w * 0.5 + 0.5;
  var shadow = 1.0;
  if shadow_uv.x >= 0.0 && shadow_uv.x <= 1.0 && shadow_uv.y >= 0.0 && shadow_uv.y <= 1.0 {
    shadow = textureSampleCompare(shadow_map, shadow_sampler, shadow_uv, shadow_z);
  }

  let radiance = uniforms.light_color * uniforms.light_intensity;
  var lo = (kd * diffuse_term + specular * radiance) * ndotl * shadow;
  lo += uniforms.base_color * uniforms.ambient;

  return vec4<f32>(lo, 1.0);
}
