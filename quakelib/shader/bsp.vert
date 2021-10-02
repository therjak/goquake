// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
uniform mat4 projection;
uniform mat4 modelview;
layout(location = 0) in vec4 Vert;
layout(location = 1) in vec2 TexCoords;
layout(location = 2) in vec2 LMCoords;
out float FogFragCoord;
out vec2 tc_tex;
out vec2 tc_lm;

void main() {
  tc_tex = TexCoords;
  tc_lm = LMCoords;
  gl_Position = projection * modelview * Vert;
  FogFragCoord = gl_Position.w;
}
