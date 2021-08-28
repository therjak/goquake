// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
uniform float Blend;
uniform vec3 ShadeVector;
uniform vec4 LightColor;
uniform mat4 projection;
uniform mat4 modelview;
layout(location = 0) in vec4 Pose1Vert;
layout(location = 1) in vec3 Pose1Normal;
layout(location = 2) in vec4 Pose2Vert;
layout(location = 3) in vec3 Pose2Normal;
layout(location = 4) in vec4 TexCoords;  // only xy are used
out float FogFragCoord;
out vec2 glTexCoord;
out vec4 frontColor;

float r_avertexnormal_dot(vec3 vertexnormal) {
  float dot = dot(vertexnormal, ShadeVector);
  // wtf - this reproduces anorm_dots within as reasonable a degree of tolerance
  // as the >= 0 case
  if (dot < 0.0)
    return 1.0 + dot * (13.0 / 44.0);
  else
    return 1.0 + dot;
}

void main() {
  glTexCoord = TexCoords.xy;
  vec4 lerpedVert =
      mix(vec4(Pose1Vert.xyz, 1.0), vec4(Pose2Vert.xyz, 1.0), Blend);
  gl_Position = projection * modelview * lerpedVert;
  FogFragCoord = gl_Position.w;
  float dot1 = r_avertexnormal_dot(Pose1Normal);
  float dot2 = r_avertexnormal_dot(Pose2Normal);
  frontColor = LightColor * vec4(vec3(mix(dot1, dot2, Blend)), 1.0);
}
