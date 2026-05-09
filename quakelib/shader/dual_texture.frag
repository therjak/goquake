// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec3 WorldPos;
out vec4 frag_color;
uniform sampler2D solid;
uniform sampler2D alpha;

uniform vec3 viewOrg;
uniform float sc1;
uniform float sc2;

void main() {
  vec3 dir = WorldPos - viewOrg;
  dir.z *= 3.0; // flatten the sphere
  float l = length(dir);
  l = (6.0 * 63.0) / l;

  vec2 st1 = (sc1 + dir.xy * l) / 128.0;
  vec2 st2 = (sc2 + dir.xy * l) / 128.0;

  vec4 color1 = texture(solid, st1);
  vec4 color2 = texture(alpha, st2);
  // TODO: add fog
  // Blend vec4(Fog_GetColor, skyfog)
  frag_color = mix(color1, color2, color2.a);
}
