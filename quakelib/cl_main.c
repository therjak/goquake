// cl_main.c  -- client main loop

#include "quakedef.h"

#include "dlight.h"

// we need to declare some mouse variables here, because the menu system
// references them even when on a unix system.

client_state_t cl;

// FIXME: put these on hunk?
efrag_t cl_efrags[MAX_EFRAGS];
entity_t cl_static_entities[MAX_STATIC_ENTITIES];
dlight_t cl_dlights[MAX_DLIGHTS];

entity_t *cl_entities;  // johnfitz -- was a static array, now on hunk

int cl_numvisedicts;
entity_t *cl_visedicts[MAX_VISEDICTS];
entity_t cl_viewent;  // the gun model

extern cvar_t r_lerpmodels, r_lerpmove;  // johnfitz

cvar_t chase_active;

/*
==============
Chase_Init
==============
*/
void Chase_Init(void) {
  Cvar_FakeRegister(&chase_active, "chase_active");
}

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
  memset(cl_efrags, 0, sizeof(cl_efrags));
  memset(cl_beams, 0, sizeof(cl_beams));
  CL_ClearDLights();

  int cl_max_edicts =
      CLAMP(MIN_EDICTS, (int)Cvar_GetValue(&max_edicts), MAX_EDICTS);
  cl_entities = (entity_t *)Hunk_AllocName(cl_max_edicts * sizeof(entity_t),
                                           "cl_entities");
  CL_SetMaxEdicts(cl_max_edicts);

  //
  // allocate the efrags and chain together into a free list
  //
  cl.free_efrags = cl_efrags;
  for (i = 0; i < MAX_EFRAGS - 1; i++)
    cl.free_efrags[i].entnext = &cl.free_efrags[i + 1];
  cl.free_efrags[i].entnext = NULL;
}

void SetCLWeaponModel(int v) {
  entity_t *view;
  view = &cl_viewent;
  view->model = cl.model_precache[v];
}

void CLPrecacheModelClear(void) {
  memset(cl.model_precache, 0, sizeof(cl.model_precache));
}
