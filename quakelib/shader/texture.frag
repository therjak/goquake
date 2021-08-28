// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec2 Texcoord;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
  vec4 color = texture(tex, Texcoord);
  if (color.a < 0.666)
    discard;
  frag_color = color;
}
