// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec2 Texcoord;
out vec4 frag_color;

// Raw palette index texture (GL_R8, normalized 0..1)
uniform sampler2D indexTex;
// Full 256-entry RGBA palette as a 1D texture
uniform sampler1D palette;
// Static 256x16 2D remap LUT texture (GL_R8)
uniform sampler2D translation;
uniform int topColor;
uniform int bottomColor;

void main() {
  // Sample the raw palette index (stored in red channel, 0..255 normalised to 0..1)
  float rawIndex = texture(indexTex, Texcoord).r;
  int uIdx = int(rawIndex * 255.0 + 0.5);

  // Determine color selection (top for 144..159, bottom for 160..175)
  float choice = (uIdx >= 160 && uIdx <= 175) ? float(bottomColor) : float(topColor);
  vec2 lutCoord = vec2(rawIndex, (choice + 0.5) / 16.0);

  // Look up the remapped index in the static 2D LUT
  float remapped = texture(translation, lutCoord).r;

  // Fetch final RGBA color from palette
  vec4 color = texture(palette, remapped);

  if (color.a < 0.666)
    discard;
  frag_color = color;
}
