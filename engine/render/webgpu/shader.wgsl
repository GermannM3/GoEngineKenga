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
  emissive_color: vec3<f32>,
  emissive_strength: f32,
  normal_scale: f32,
  alpha_cutoff: f32,
  flags: u32,
  point_light_pos: vec3<f32>,
  point_light_intensity: f32,
  point_light_color: vec3<f32>,
  point_light_range: f32,
}

@group(0) @binding(0)
var<uniform> uniforms: Uniforms;

@group(1) @binding(0)
var shadow_map: texture_depth_2d;
@group(1) @binding(1)
var shadow_sampler: sampler_comparison;

// Текстуры материала (группа 2): color, normal, metallic-roughness, emissive + общий sampler.
// Пустые слоты биндятся fallback-текстурами (белая/плоская/чёрная) со стороны Go.
@group(2) @binding(0)
var color_tex: texture_2d<f32>;
@group(2) @binding(1)
var color_sampler: sampler;
@group(2) @binding(2)
var normal_tex: texture_2d<f32>;
@group(2) @binding(3)
var normal_sampler: sampler;
@group(2) @binding(4)
var mr_tex: texture_2d<f32>;
@group(2) @binding(5)
var mr_sampler: sampler;
@group(2) @binding(6)
var emissive_tex: texture_2d<f32>;
@group(2) @binding(7)
var emissive_sampler: sampler;

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
  // Альбедо из текстуры (fallback-белая даёт чистый uniforms.base_color)
  let base_tex = textureSample(color_tex, color_sampler, in.uv);
  let albedo = base_tex.rgb * uniforms.base_color;
  let alpha = base_tex.a;

  // Normal mapping через производные экрана (в вершинном буфере нет tangents)
  let n_geometric = normalize(in.world_normal);
  let dp1 = dpdx(in.world_pos);
  let dp2 = dpdy(in.world_pos);
  let duv1 = dpdx(in.uv);
  let duv2 = dpdy(in.uv);
  let t = normalize(dp1 * duv2.y - dp2 * duv1.y);
  let b = normalize(cross(n_geometric, t));
  let n_tex = textureSample(normal_tex, normal_sampler, in.uv).rgb * 2.0 - 1.0;
  let n_scaled = n_tex * vec3<f32>(uniforms.normal_scale, uniforms.normal_scale, 1.0);
  let n = normalize(t * n_scaled.x + b * n_scaled.y + n_geometric * n_scaled.z);

  // Metallic/roughness: из текстуры (G/R), scaled факторами; флаг — текстура задана
  var metallic = uniforms.metallic;
  var roughness = uniforms.roughness;
  if (uniforms.flags & 1u) != 0u {
    let mr = textureSample(mr_tex, mr_sampler, in.uv);
    metallic = mr.b * uniforms.metallic;
    roughness = max(mr.g * uniforms.roughness, 0.04);
  }

  // AlphaMode MASK: отбрасываем пиксели ниже cutoff
  if uniforms.alpha_cutoff > 0.0 && alpha < uniforms.alpha_cutoff {
    discard;
  }

  let v = normalize(uniforms.cam_pos - in.world_pos);
  let l = normalize(uniforms.light_dir);
  let h = normalize(v + l);

  let ndotv = max(dot(n, v), 0.0001);
  let ndotl = max(dot(n, l), 0.0);

  // F0 for dielectric/metallic blend
  let f0 = mix(vec3<f32>(0.04), albedo, metallic);

  // Diffuse (Lambert)
  let diffuse = albedo * (1.0 - metallic);
  let diffuse_term = diffuse * ndotl;

  // Specular (Cook-Torrance simplified)
  let d = distribution_ggx(n, h, max(roughness, 0.04));
  let f = fresnel_schlick(max(dot(h, v), 0.0), f0);
  let kd = (1.0 - f) * (1.0 - metallic);
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

  // Point light: distance-based attenuation, без собственных теней (cubemap shadows — в backlog).
  if uniforms.point_light_intensity > 0.0 {
    let pl_dir = uniforms.point_light_pos - in.world_pos;
    let pl_dist = length(pl_dir);
    let pl_l = pl_dir / max(pl_dist, 0.0001);
    let pl_norm = pl_dist / max(uniforms.point_light_range, 0.0001);
    let attenuation = 1.0 / (1.0 + pl_norm * pl_norm);
    let pl_h = normalize(v + pl_l);
    let pl_ndotl = max(dot(n, pl_l), 0.0);
    let pl_d = distribution_ggx(n, pl_h, max(roughness, 0.04));
    let pl_f = fresnel_schlick(max(dot(pl_h, v), 0.0), f0);
    let pl_kd = (1.0 - pl_f) * (1.0 - metallic);
    let pl_specular = pl_f * pl_d * 0.25;
    let pl_radiance = uniforms.point_light_color * uniforms.point_light_intensity * attenuation;
    lo += (pl_kd * diffuse * pl_ndotl + pl_specular * pl_radiance) * pl_ndotl;
  }

  // Ambient
  lo += albedo * uniforms.ambient;

  // Emissive (fallback-чёрная текстура даёт 0)
  let emissive = textureSample(emissive_tex, emissive_sampler, in.uv).rgb * uniforms.emissive_color * uniforms.emissive_strength;
  lo += emissive;

  return vec4<f32>(lo, alpha);
}
