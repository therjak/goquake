// SPDX-License-Identifier: GPL-2.0-or-later
#include "_cgo_export.h"
#include "quakedef.h"

void CLPrecacheModel(const char* cn, int i) {
  if (i != 1) {
    return;
  }
  cl.worldmodel = Mod_ForName(cn);
}

void FinishCL_ParseServerInfo(void) {
  // local state
  Hunk_Check();  // make sure nothing is hurt
}
