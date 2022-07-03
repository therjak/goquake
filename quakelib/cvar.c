// SPDX-License-Identifier: GPL-2.0-or-later
#include "quakedef.h"

float Cvar_GetValue(cvar_t *variable) {
  return CvarGetValue(variable->id);
}

void Cvar_FakeRegister(cvar_t *v, char *name) {
  v->id = CvarGetID(name);
}
