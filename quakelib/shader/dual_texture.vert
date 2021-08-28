// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 position;
layout(location = 1) in vec2 solidtexcoord;
layout(location = 2) in vec2 alphatexcoord;
out vec2 SolidTexcoord;
out vec2 AlphaTexcoord;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  SolidTexcoord = solidtexcoord;
  AlphaTexcoord = alphatexcoord;
  gl_Position = projection * modelview * vec4(position, 1.0);
}
