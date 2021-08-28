// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 position;
layout(location = 1) in vec2 texcoord;
out vec2 Texcoord;

void main() {
  Texcoord = texcoord;
  gl_Position = vec4(position, 1.0);
}
