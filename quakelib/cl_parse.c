// SPDX-License-Identifier: GPL-2.0-or-later
#include "_cgo_export.h"
#include "quakedef.h"

void CLPrecacheModel(const char* cn, int i) {
  cl.model_precache[i] = Mod_ForName(cn);
}

void FinishCL_ParseServerInfo(void) {
  // local state
  cl.worldmodel = cl.model_precache[1];

  Hunk_Check();              // make sure nothing is hurt
  noclip_anglehack = false;  // noclip is turned off at start
}
