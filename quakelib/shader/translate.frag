// SPDX-License-Identifier: GPL-2.0-or-later
#version 330
in vec2 Texcoord;
out vec4 frag_color;

// Raw palette index texture (GL_R8UI via GL_RED + GL_UNSIGNED_BYTE)
uniform sampler2D indexTex;
// Full 256-entry RGBA palette as a 1D texture
uniform sampler1D palette;
// 256-entry remap table: translation[i] -> remapped palette index
uniform sampler1D translation;

void main() {
  // Sample the raw palette index (stored in red channel, 0..255 normalised to 0..1)
  float rawIndex = texture(indexTex, Texcoord).r;

  // Look up the remapped index through the translation LUT.
  // Both 1D textures are 256 texels wide; sample at the centre of each texel.
  float remapped = texture(translation, rawIndex).r;

  // Fetch the final RGBA colour from the palette.
  vec4 color = texture(palette, remapped);

  if (color.a < 0.666)
    discard;
  frag_color = color;
}
