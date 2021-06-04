// SPDX-License-Identifier: GPL-2.0-or-later
#include "_cgo_export.h"
#include "quakedef.h"

entity_t cl_temp_entities[MAX_TEMP_ENTITIES];

void CLPrecacheModel(const char* cn, int i) {
  cl.model_precache[i] = Mod_ForName(cn);
}

void FinishCL_ParseServerInfo(void) {
  // local state
  cl.worldmodel = cl.model_precache[1];
  SetWorldEntityModel(cl.worldmodel);

  R_NewMap();

  Hunk_Check();              // make sure nothing is hurt
  noclip_anglehack = false;  // noclip is turned off at start
}

void CL_ParseUpdate(int num, int modnum) {
  qmodel_t* model;
  entity_t* ent;
  ent = CL_EntityNum(num);
  model = cl.model_precache[modnum];
  if (model != ent->model) {
    ent->model = model;
  }
}

void CL_ParseStaticC(entity_t* ent, int modelindex) {
  ent->model = cl.model_precache[modelindex];
}
