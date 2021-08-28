// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 vposition;
layout(location = 1) in vec2 vtexcoord;
layout(location = 2) in vec3 vcolor;
out vec2 Texcoord;
out vec3 InColor;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  Texcoord = vtexcoord;
  InColor = vcolor;
  gl_Position = projection * modelview * vec4(vposition, 1.0);
}
