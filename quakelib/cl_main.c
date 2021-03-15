// SPDX-License-Identifier: GPL-2.0-or-later
// cl_main.c  -- client main loop

#include "quakedef.h"

#include "dlight.h"

// we need to declare some mouse variables here, because the menu system
// references them even when on a unix system.

client_state_t cl;

// FIXME: put these on hunk?
entity_t cl_static_entities[MAX_STATIC_ENTITIES];
dlight_t cl_dlights[MAX_DLIGHTS];

entity_t *cl_entities;  // johnfitz -- was a static array, now on hunk

entity_t cl_viewent;  // the gun model

/*
=====================
CL_ClearState

=====================
*/
void CL_ClearState(void) {
  int i;

  if (!SV_Active()) Host_ClearMemory();

  // wipe the entire cl structure
  memset(&cl, 0, sizeof(cl));
  CL_Clear();  // and on the go side

  CLSMessageClear();

  // clear other arrays
  CL_ClearDLights();

  int cl_max_edicts =
      CLAMP(MIN_EDICTS, (int)Cvar_GetValue(&max_edicts), MAX_EDICTS);
  cl_entities = (entity_t *)Hunk_AllocName(cl_max_edicts * sizeof(entity_t),
                                           "cl_entities");
  CL_SetMaxEdicts(cl_max_edicts);
}

void SetCLWeaponModel(int v) {
  entity_t *view;
  view = &cl_viewent;
  view->model = cl.model_precache[v];
}

void CLPrecacheModelClear(void) {
  memset(cl.model_precache, 0, sizeof(cl.model_precache));
}
