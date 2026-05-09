// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 position;
out vec3 WorldPos;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  WorldPos = position;
  gl_Position = projection * modelview * vec4(position, 1.0);
}
