// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
out vec4 frag_color;
uniform vec4 in_color;

void main() {
  frag_color = in_color;
}
