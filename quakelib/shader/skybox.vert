// SPDX-License-Identifier: GPL-2.0-or-later
#version 330 core
layout(location = 0) in vec3 position;
uniform mat4 projection;
uniform mat4 modelview;
out vec3 glTexCoord;

void main() {
  glTexCoord = position;
  vec4 p = projection * modelview * vec4(position, 1.0);
  gl_Position = p.xyww;
}
