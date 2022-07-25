// SPDX-License-Identifier: GPL-2.0-or-later
#version 330 core
uniform samplerCube skybox;
in vec3 glTexCoord;
out vec4 frag_color;

void main() {
  frag_color = texture(skybox, glTexCoord);
}
