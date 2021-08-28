// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec2 Texcoord;
out vec4 frag_color;
uniform sampler2D tex;
uniform float gamma;
uniform float contrast;

void main() {
  vec4 color = texture(tex, Texcoord);
  color.rgb = color.rgb * contrast;
  frag_color = vec4(pow(color.rgb, vec3(gamma)), 1.0);
}
