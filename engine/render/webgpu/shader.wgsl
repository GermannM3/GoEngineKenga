// GoEngineKenga WebGPU PBR shader
// Vertex: position, normal, uv
// Instance: model matrix (locations 3-6)
// Uniform: viewProj, Material, Light

struct VertexInput {
  @location(0) position: vec3<f32>,
  @location(1) normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
}

struct InstanceInput {
  @location(3) model_0: vec4<f32>,
  @location(4) model_1: vec4<f32>,
  @location(5) model_2: vec4<f32>,
  @location(6) model_3: vec4<f32>,
}

struct VertexOutput {
  @builtin(position) clip_position: vec4<f32>,
  @location(0) world_pos: vec3<f32>,
  @location(1) world_normal: vec3<f32>,
  @location(2) uv: vec2<f32>,
}

struct Uniforms {
  view_proj: mat4x4<f32>,
  _model_pad: mat4x4<f32>,
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

@group(0) @binding(0)
var<uniform> uniforms: Uniforms;

@group(1) @binding(0)
var shadow_map: texture_depth_2d;
@group(1) @binding(1)
var shadow_sampler: sampler_comparison;

@vertex
fn vs_main(in: VertexInput, instance: InstanceInput) -> VertexOutput {
  let model = mat4x4<f32>(instance.model_0, instance.model_1, instance.model_2, instance.model_3);
  let mvp = uniforms.view_proj * model;
  var out: VertexOutput;
  out.clip_position = mvp * vec4<f32>(in.position, 1.0);
  out.world_pos = (model * vec4<f32>(in.position, 1.0)).xyz;
  out.world_normal = (model * vec4<f32>(in.normal, 0.0)).xyz;
  out.uv = in.uv;
  return out;
}

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

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
  let n = normalize(in.world_normal);
  let v = normalize(uniforms.cam_pos - in.world_pos);
  let l = normalize(uniforms.light_dir);
  let h = normalize(v + l);

  let ndotv = max(dot(n, v), 0.0001);
  let ndotl = max(dot(n, l), 0.0);

  // F0 for dielectric/metallic blend
  let f0 = mix(vec3<f32>(0.04), uniforms.base_color, uniforms.metallic);

  // Diffuse (Lambert)
  let diffuse = uniforms.base_color * (1.0 - uniforms.metallic);
  let diffuse_term = diffuse * ndotl;

  // Specular (Cook-Torrance simplified)
  let d = distribution_ggx(n, h, max(uniforms.roughness, 0.04));
  let f = fresnel_schlick(max(dot(h, v), 0.0), f0);
  let kd = (1.0 - f) * (1.0 - uniforms.metallic);
  let specular = f * d * 0.25;

  // Shadow: project world pos to light space
  let light_clip = uniforms.light_view_proj * vec4<f32>(in.world_pos, 1.0);
  let shadow_z = light_clip.z / light_clip.w;
  let shadow_uv = light_clip.xy / light_clip.w * 0.5 + 0.5;
  var shadow = 1.0;
  if shadow_uv.x >= 0.0 && shadow_uv.x <= 1.0 && shadow_uv.y >= 0.0 && shadow_uv.y <= 1.0 {
    shadow = textureSampleCompare(shadow_map, shadow_sampler, shadow_uv, shadow_z);
  }

  let radiance = uniforms.light_color * uniforms.light_intensity;
  var lo = (kd * diffuse_term + specular * radiance) * ndotl * shadow;

  // Ambient
  lo += uniforms.base_color * uniforms.ambient;

  return vec4<f32>(lo, 1.0);
}
