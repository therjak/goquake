// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
uniform sampler2D Tex;
uniform sampler2D LMTex;
uniform sampler2D FullbrightTex;
uniform bool UseFullbrightTex;
uniform bool UseOverbright;
uniform bool UseAlphaTest;
uniform float Alpha;
uniform float FogDensity;
uniform vec4 FogColor;
in float FogFragCoord;
in vec2 tc_tex;
in vec2 tc_lm;
out vec4 frag_color;

void main() {
  vec4 result = texture2D(Tex, tc_tex.xy);
  if (UseAlphaTest && result.a < 0.666)
    discard;
  result *= texture2D(LMTex, tc_lm.xy);
  if (UseOverbright)
    result.rgb *= 2.0;
  if (UseFullbrightTex)
    result += texture2D(FullbrightTex, tc_tex.xy);
  result = clamp(result, 0.0, 1.0);
  float fog = exp(-FogDensity * FogDensity * FogFragCoord * FogFragCoord);
  fog = clamp(fog, 0.0, 1.0);
  result = mix(FogColor, result, fog);
  result.a = Alpha;
  frag_color = result;
}
