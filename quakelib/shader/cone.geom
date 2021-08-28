// SPDX-License-Identifier: GPL-2.0-or-later
#version 330 core
layout(points) in;
layout(triangle_strip, max_vertices = 32) out;  // 8*4 vertices

const float M_S45 = 0.70710678118;
const float M_S225 = 0.38268343236;
const float M_S675 = 0.92387953251;

in VS_OUT {
  vec3 radius;
  vec3 innerColor;
  vec3 outerColor;
}
gs_in[];

out vec3 Color;

void buildCone(vec4 position) {
  vec3 r = gs_in[0].radius;
  vec4 front = position - vec4(0.0, 0.0, r.z, 0.0);
  vec2 p1 = vec2(r.x * M_S675, r.y * M_S225);
  vec2 p2 = vec2(r.x * M_S45, r.y * M_S45);
  vec2 p3 = vec2(r.x * M_S225, r.y * M_S675);

  Color = gs_in[0].outerColor;
  // Part 1
  gl_Position = position + vec4(r.x, 0.0, 0.0, 0.0);
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(p1.x, p1.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(p2.x, p2.y, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 2
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(p3.x, p3.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(0.0, r.y, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 3
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(-p3.x, p3.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(-p2.x, p2.y, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 4
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(-p1.x, p1.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(-r.x, 0, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 5
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(-p1.x, -p1.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(-p2.x, -p2.y, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 6
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(-p3.x, -p3.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(0.0, -r.y, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 7
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(p3.x, -p3.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(p2.x, -p2.y, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();

  // Part 8
  EmitVertex();

  Color = gs_in[0].innerColor;
  gl_Position = front;
  EmitVertex();
  Color = gs_in[0].outerColor;

  gl_Position = position + vec4(p1.x, -p1.y, 0.0, 0.0);
  EmitVertex();

  gl_Position = position + vec4(r.x, 0, 0.0, 0.0);
  EmitVertex();

  EndPrimitive();
}

void main() {
  buildCone(gl_in[0].gl_Position);
}
