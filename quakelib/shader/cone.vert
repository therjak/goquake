// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
layout(location = 0) in vec3 position;
layout(location = 1) in float radius;
layout(location = 2) in vec3 innerColor;
layout(location = 3) in vec3 outerColor;
out VS_OUT {
  vec3 radius;
  vec3 innerColor;
  vec3 outerColor;
}
vs_out;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
  vs_out.innerColor = innerColor;
  vs_out.outerColor = outerColor;
  vec4 mc = modelview * vec4(position, 1.0);
  float x = projection[0][0] * (mc.x + radius);
  float y = projection[1][1] * (mc.y + radius);
  gl_Position = projection * mc;
  vs_out.radius.x = abs(gl_Position.x - x);
  vs_out.radius.y = abs(gl_Position.y - y);
  // No need to consider the frustum for the z radius.
  vs_out.radius.z = radius / gl_Position.w;
}
