// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 position;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  gl_Position = projection * modelview * vec4(position, 1.0);
}
