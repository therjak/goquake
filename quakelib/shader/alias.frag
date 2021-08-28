// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
uniform sampler2D Tex;
uniform sampler2D FullbrightTex;
uniform bool UseFullbrightTex;
uniform bool UseOverbright;
uniform float FogDensity;
uniform vec4 FogColor;
in float FogFragCoord;
in vec2 glTexCoord;
in vec4 frontColor;
out vec4 frag_color;

void main() {
  vec4 result = texture2D(Tex, glTexCoord);
  result *= frontColor;
  if (UseOverbright)
    result.rgb *= 2.0;
  if (UseFullbrightTex)
    result += texture2D(FullbrightTex, glTexCoord.xy);
  result = clamp(result, 0.0, 1.0);
  float fog = exp(-FogDensity * FogDensity * FogFragCoord * FogFragCoord);
  fog = clamp(fog, 0.0, 1.0);
  result = mix(FogColor, result, fog);
  result.a = frontColor.a;
  frag_color = result;
}
