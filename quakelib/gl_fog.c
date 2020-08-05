#include "quakedef.h"

float fogColor[4];

void Fog_SetupFrame(void) {
  glFogfv(GL_FOG_COLOR, Fog_GetColor());
  glFogf(GL_FOG_DENSITY, Fog_GetDensity() / 64.0);
}

void Fog_EnableGFog(void) {
  if (Fog_GetDensity() > 0) glEnable(GL_FOG);
}

void Fog_DisableGFog(void) {
  if (Fog_GetDensity() > 0) glDisable(GL_FOG);
}

void Fog_SetupState(void) {
  glFogi(GL_FOG_MODE, GL_EXP2);
}

void Fog_Init(void) {
  Fog_SetupState();
}

