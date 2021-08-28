// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec2 SolidTexcoord;
in vec2 AlphaTexcoord;
out vec4 frag_color;
uniform sampler2D solid;
uniform sampler2D alpha;

void main() {
  vec4 color1 = texture(solid, SolidTexcoord);
  vec4 color2 = texture(alpha, AlphaTexcoord);
  // TODO: add fog
  // Blend vec4(Fog_GetColor, skyfog)
  frag_color = mix(color1, color2, color2.a);
}
