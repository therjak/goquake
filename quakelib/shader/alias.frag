// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
uniform sampler2D Tex;
uniform sampler2D FullbrightTex;
uniform bool UseFullbrightTex;
uniform bool UseOverbright;
uniform float FogDensity;
uniform vec4 FogColor;

uniform sampler1D palette;
uniform sampler2D translation;
uniform bool UseTranslation;
uniform int topColor;
uniform int bottomColor;

in float FogFragCoord;
in vec2 glTexCoord;
in vec4 frontColor;
out vec4 frag_color;

void main() {
  vec4 texColor;
  if (UseTranslation) {
    float rawIndex = texture(Tex, glTexCoord).r;
    int uIdx = int(rawIndex * 255.0 + 0.5);
    float choice = (uIdx >= 160 && uIdx <= 175) ? float(bottomColor) : float(topColor);
    vec2 lutCoord = vec2(rawIndex, (choice + 0.5) / 16.0);
    float remapped = texture(translation, lutCoord).r;
    texColor = texture(palette, remapped);
  } else {
    texColor = texture(Tex, glTexCoord);
  }

  vec4 result = texColor * frontColor;
  if (UseOverbright)
    result.rgb *= 2.0;
  if (UseFullbrightTex)
    result += texture(FullbrightTex, glTexCoord.xy);
  result = clamp(result, 0.0, 1.0);
  float fog = exp(-FogDensity * FogDensity * FogFragCoord * FogFragCoord);
  fog = clamp(fog, 0.0, 1.0);
  result = mix(FogColor, result, fog);
  result.a = frontColor.a;
  frag_color = result;
}
