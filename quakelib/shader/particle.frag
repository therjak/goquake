// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec2 Texcoord;
in vec3 InColor;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
  float color = texture(tex, Texcoord).r;
  frag_color.rgb = InColor;
  frag_color.a = color;  // texture has only one chan
  frag_color = clamp(frag_color, vec4(0, 0, 0, 0), vec4(1, 1, 1, 1));
}
