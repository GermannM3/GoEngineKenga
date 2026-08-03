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
  emissive_color: vec3<f32>,
  emissive_strength: f32,
  normal_scale: f32,
  alpha_cutoff: f32,
  flags: u32,
  point_light_pos: vec3<f32>,
  point_light_intensity: f32,
  point_light_color: vec3<f32>,
  point_light_range: f32,
  spot_light_pos: vec3<f32>,
  spot_light_dir: vec3<f32>,
  spot_light_color: vec3<f32>,
  spot_light_intensity: f32,
  spot_light_range: f32,
  spot_inner_cos: f32,
  spot_outer_cos: f32,
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
@group(1) @binding(2)
var shadow_map_point: texture_depth_2d_array;
@group(1) @binding(3)
var shadow_sampler_point: sampler_comparison;

// Текстуры материала (группа 2): color, normal, metallic-roughness, emissive + общий sampler.
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

// IBL окружение (группа 3): env-кубомапа + irradiance-кубомапа.
@group(3) @binding(0)
var env_tex: texture_cube<f32>;
@group(3) @binding(1)
var env_sampler: sampler;
@group(3) @binding(2)
var irr_tex: texture_cube<f32>;
@group(3) @binding(3)
var irr_sampler: sampler;

fn fresnel_schlick(cos_theta: f32, f0: vec3<f32>) -> vec3<f32> {
  return f0 + (1.0 - f0) * pow(1.0 - cos_theta, 5.0);
}

// ACES filmic tonemap (Narkowicz 2015): переводит HDR-свет в LDR без
// пережжённых бликов и «мутного» серого.
fn aces(x: vec3<f32>) -> vec3<f32> {
  let a = 2.51;
  let b = 0.03;
  let c = 2.43;
  let d = 0.59;
  let e = 0.14;
  return clamp((x * (a * x + b)) / (x * (c * x + d) + e), vec3<f32>(0.0), vec3<f32>(1.0));
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

  let f0 = mix(vec3<f32>(0.04), albedo, metallic);
  let diffuse = albedo * (1.0 - metallic);
  let diffuse_term = diffuse * ndotl;
  let d = distribution_ggx(n, h, max(roughness, 0.04));
  let f = fresnel_schlick(max(dot(h, v), 0.0), f0);
  let kd = (1.0 - f) * (1.0 - metallic);
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
    // Cubemap-тени point light: 6 граней в depth_2d_array, линейная глубина (dist/range).
    let sd = in.world_pos - uniforms.point_light_pos;
    let sa = abs(sd);
    let sm = max(sa.x, max(sa.y, sa.z));
    var sl: u32 = 0u;
    var suv = vec2<f32>(0.0);
    if (sa.x >= sa.y && sa.x >= sa.z) {
      if (sd.x >= 0.0) { sl = 0u; suv = vec2<f32>(-sd.z, -sd.y) / sa.x; }
      else { sl = 1u; suv = vec2<f32>(sd.z, -sd.y) / sa.x; }
    } else if (sa.y >= sa.x && sa.y >= sa.z) {
      if (sd.y >= 0.0) { sl = 2u; suv = vec2<f32>(sd.x, sd.z) / sa.y; }
      else { sl = 3u; suv = vec2<f32>(sd.x, -sd.z) / sa.y; }
    } else {
      if (sd.z >= 0.0) { sl = 4u; suv = vec2<f32>(sd.x, -sd.y) / sa.z; }
      else { sl = 5u; suv = vec2<f32>(-sd.x, -sd.y) / sa.z; }
    }
    let pl_uv = suv * 0.5 + 0.5;
    let pl_depth = sm / max(uniforms.point_light_range, 0.0001);
    var pl_shadow = 1.0;
    if pl_uv.x >= 0.0 && pl_uv.x <= 1.0 && pl_uv.y >= 0.0 && pl_uv.y <= 1.0 {
      pl_shadow = textureSampleCompare(shadow_map_point, shadow_sampler_point, pl_uv, sl, pl_depth - 0.001);
    }
    lo += (pl_kd * diffuse * pl_ndotl + pl_specular * pl_radiance) * pl_ndotl * pl_shadow;
  }

  // Spotlight: конус (inner/outer) + затухание по расстоянию, без собственных теней (backlog).
  if uniforms.spot_light_intensity > 0.0 {
    let sl_dir = uniforms.spot_light_pos - in.world_pos;
    let sl_dist = length(sl_dir);
    let sl_l = sl_dir / max(sl_dist, 0.0001);
    let sl_norm = sl_dist / max(uniforms.spot_light_range, 0.0001);
    let sl_att = 1.0 / (1.0 + sl_norm * sl_norm);
    let sl_cone = smoothstep(uniforms.spot_outer_cos, uniforms.spot_inner_cos, dot(-sl_l, normalize(uniforms.spot_light_dir)));
    if sl_cone > 0.0 {
      let sl_h = normalize(v + sl_l);
      let sl_ndotl = max(dot(n, sl_l), 0.0);
      let sl_d = distribution_ggx(n, sl_h, max(roughness, 0.04));
      let sl_f = fresnel_schlick(max(dot(sl_h, v), 0.0), f0);
      let sl_kd = (1.0 - sl_f) * (1.0 - metallic);
      let sl_specular = sl_f * sl_d * 0.25;
      let sl_radiance = uniforms.spot_light_color * uniforms.spot_light_intensity * sl_att * sl_cone;
      lo += (sl_kd * diffuse * sl_ndotl + sl_specular * sl_radiance) * sl_ndotl;
    }
  }

  lo += albedo * uniforms.ambient;

  // IBL: диффузный свет из irradiance-кубомапы + грубое отражение окружения.
  let irr_env = textureSample(irr_tex, irr_sampler, n).rgb;
  lo += albedo * irr_env * 0.5;
  let r_env = reflect(-v, n);
  let env_ref = textureSample(env_tex, env_sampler, r_env).rgb;
  lo += env_ref * (fresnel_schlick(max(dot(n, v), 0.0), f0) * (1.0 - roughness) * 0.4);

  // Emissive (fallback-чёрная текстура даёт 0)
  let emissive = textureSample(emissive_tex, emissive_sampler, in.uv).rgb * uniforms.emissive_color * uniforms.emissive_strength;
  lo += emissive;

  return vec4<f32>(aces(lo), alpha);
}
