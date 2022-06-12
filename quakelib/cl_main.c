// SPDX-License-Identifier: GPL-2.0-or-later
// cl_main.c  -- client main loop

#include "dlight.h"
#include "quakedef.h"

// we need to declare some mouse variables here, because the menu system
// references them even when on a unix system.

client_state_t cl;

// FIXME: put these on hunk?
dlight_t cl_dlights[MAX_DLIGHTS];

/*
=====================
CL_ClearState

=====================
*/
void CL_ClearState(void) {
  // wipe the entire cl structure
  memset(&cl, 0, sizeof(cl));
}

void CLPrecacheModelClear(void) {
  memset(cl.model_precache, 0, sizeof(cl.model_precache));
}
