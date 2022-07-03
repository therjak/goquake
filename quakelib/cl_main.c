// SPDX-License-Identifier: GPL-2.0-or-later
// cl_main.c  -- client main loop

#include "quakedef.h"

// we need to declare some mouse variables here, because the menu system
// references them even when on a unix system.

client_state_t cl;

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
}
