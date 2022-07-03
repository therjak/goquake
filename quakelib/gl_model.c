// SPDX-License-Identifier: GPL-2.0-or-later

#include "quakedef.h"

/*
===============
Mod_Init
===============
*/
void Mod_Init(void) {
  Cvar_FakeRegister(&gl_subdivide_size, "gl_subdivide_size");
}
