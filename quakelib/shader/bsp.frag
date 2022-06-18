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
uniform bool Turb;
uniform float Time;
in float FogFragCoord;
in vec2 tc_tex;
in vec2 tc_lm;
out vec4 frag_color;

vec2 texPos() {
  float s = tc_tex.x;
  float t = tc_tex.y;
  if (!Turb) {
    return vec2(s, t);
  }
  const float freq = 0.8f;
  const float amp = 0.2f;
  const float timeScale = 1.0f;
  const float scale = 2.f;
  s *= scale;
  t *= scale;

  float posX = (2 * (t - 1.0f) + Time * timeScale) * freq;
  float posY = (2 * s + Time * timeScale) * freq;

  float texX = sin(posX) * amp;
  float texY = sin(posY) * amp;

  return vec2(texX, texY) + vec2(s, t);
}

void main() {
  vec2 texPos = texPos();
  vec4 result = texture2D(Tex, texPos);
  if (UseAlphaTest && result.a < 0.666)
    discard;
  // result *= texture2D(LMTex, tc_lm.xy);
  if (UseOverbright)
    result.rgb *= 2.0;
  if (UseFullbrightTex)
    result += texture2D(FullbrightTex, texPos);
  result = clamp(result, 0.0, 1.0);
  float fog = exp(-FogDensity * FogDensity * FogFragCoord * FogFragCoord);
  fog = clamp(fog, 0.0, 1.0);
  result = mix(FogColor, result, fog);
  result.a = Alpha;
  frag_color = result;
}
