// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec3 Color;
out vec4 frag_color;

void main() {
  frag_color = vec4(Color, 1.0);
}
