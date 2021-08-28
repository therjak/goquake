// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 position;
layout(location = 1) in vec2 texcoord;
out vec2 Texcoord;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  Texcoord = texcoord;
  gl_Position = projection * modelview * vec4(position, 1.0);
}
