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
  memset(cl_dlights, 0, sizeof(cl_dlights));
  memset(cl_temp_entities, 0, sizeof(cl_temp_entities));
  memset(cl_beams, 0, sizeof(cl_beams));

  // johnfitz -- cl_entities is now dynamically allocated
  int cl_max_edicts =
      CLAMP(MIN_EDICTS, (int)Cvar_GetValue(&max_edicts), MAX_EDICTS);
  CL_SetMaxEdicts(cl_max_edicts);
  cl_entities = (entity_t *)Hunk_AllocName(cl_max_edicts * sizeof(entity_t),
                                           "cl_entities");
  // johnfitz

  //
  // allocate the efrags and chain together into a free list
  //
  cl.free_efrags = cl_efrags;
  for (i = 0; i < MAX_EFRAGS - 1; i++)
    cl.free_efrags[i].entnext = &cl.free_efrags[i + 1];
  cl.free_efrags[i].entnext = NULL;
}

/*
==============
CL_PrintEntities_f
==============
*/
void CL_PrintEntities_f(void) {
  entity_t *ent;
  int i;

  if (CLS_GetState() != ca_connected) return;

  for (i = 0, ent = cl_entities; i < CL_num_entities(); i++, ent++) {
    Con_Printf("%3i:", i);
    if (!ent->model) {
      Con_Printf("EMPTY\n");
      continue;
    }
    Con_Printf("%s:%2i  (%5.1f,%5.1f,%5.1f) [%5.1f %5.1f %5.1f]\n",
               ent->model->name, ent->frame, ent->origin[0], ent->origin[1],
               ent->origin[2], ent->angles[0], ent->angles[1], ent->angles[2]);
  }
}

/*
===============
CL_RelinkEntities
===============
*/
void CL_RelinkEntities(void) {
  entity_t *ent;
  int i, j;
  float frac, f, d;
  vec3_t delta;
  float bobjrotate;
  vec3_t oldorg;
  dlight_t *dl;

  // determine partial update time
  frac = CL_LerpPoint();

  cl_numvisedicts = 0;

  //
  // interpolate player info
  //
  for (i = 0; i < 3; i++) {
    float v =
        CL_MVelocity(1, i) + frac * (CL_MVelocity(0, i) - CL_MVelocity(1, i));
    CL_SetVelocity(i, v);
  }

  if (CLS_IsDemoPlayback()) {
    // interpolate the angles
    {
      d = CL_MViewAngles(0, 0) - CL_MViewAngles(1, 0);
      if (d > 180)
        d -= 360;
      else if (d < -180)
        d += 360;
      SetCLPitch(CL_MViewAngles(1, 0) + frac * d);
    }
    {
      d = CL_MViewAngles(0, 1) - CL_MViewAngles(1, 1);
      if (d > 180)
        d -= 360;
      else if (d < -180)
        d += 360;
      SetCLYaw(CL_MViewAngles(1, 1) + frac * d);
    }
    {
      d = CL_MViewAngles(0, 2) - CL_MViewAngles(1, 2);
      if (d > 180)
        d -= 360;
      else if (d < -180)
        d += 360;
      SetCLRoll(CL_MViewAngles(1, 2) + frac * d);
    }
  }

  bobjrotate = anglemod(100 * CL_Time());

  // start on the entity after the world
  for (i = 1, ent = cl_entities + 1; i < CL_num_entities(); i++, ent++) {
    if (!ent->model) {                          // empty slot
      if (ent->forcelink) R_RemoveEfrags(ent);  // just became empty
      continue;
    }

    // if the object wasn't included in the last packet, remove it
    if (ent->msgtime != CL_MTime()) {
      ent->model = NULL;
      ent->lerpflags |=
          LERP_RESETMOVE | LERP_RESETANIM;  // johnfitz -- next time this entity
                                            // slot is reused, the lerp will
                                            // need to be reset
      continue;
    }

    VectorCopy(ent->origin, oldorg);

    if (ent->forcelink) {  // the entity was not updated in the last message
      // so move to the final spot
      VectorCopy(ent->msg_origins[0], ent->origin);
      VectorCopy(ent->msg_angles[0], ent->angles);
    } else {  // if the delta is large, assume a teleport and don't lerp
      f = frac;
      for (j = 0; j < 3; j++) {
        delta[j] = ent->msg_origins[0][j] - ent->msg_origins[1][j];
        if (delta[j] > 100 || delta[j] < -100) {
          f = 1;  // assume a teleportation, not a motion
          ent->lerpflags |= LERP_RESETMOVE;  // johnfitz -- don't lerp teleports
        }
      }

      // johnfitz -- don't cl_lerp entities that will be r_lerped
      if (Cvar_GetValue(&r_lerpmove) && (ent->lerpflags & LERP_MOVESTEP)) f = 1;
      // johnfitz

      // interpolate the origin and angles
      for (j = 0; j < 3; j++) {
        ent->origin[j] = ent->msg_origins[1][j] + f * delta[j];

        d = ent->msg_angles[0][j] - ent->msg_angles[1][j];
        if (d > 180)
          d -= 360;
        else if (d < -180)
          d += 360;
        ent->angles[j] = ent->msg_angles[1][j] + f * d;
      }
    }

    // rotate binary objects locally
    if (ent->model->flags & EF_ROTATE) ent->angles[1] = bobjrotate;

    if (ent->effects & EF_BRIGHTFIELD) ParticlesAddEntity(ent);

    if (ent->effects & EF_MUZZLEFLASH) {
      vec3_t fv, rv, uv;

      dl = CL_AllocDlight(i);
      VectorCopy(ent->origin, dl->origin);
      dl->origin[2] += 16;
      AngleVectors(ent->angles, fv, rv, uv);

      VectorMA(dl->origin, 18, fv, dl->origin);
      dl->radius = 200 + (rand() & 31);
      dl->minlight = 32;
      dl->die = CL_Time() + 0.1;

      // johnfitz -- assume muzzle flash accompanied by muzzle flare, which
      // looks bad when lerped
      if (Cvar_GetValue(&r_lerpmodels) != 2) {
        if (ent == CLViewEntity())
          cl_viewent.lerpflags |=
              LERP_RESETANIM | LERP_RESETANIM2;  // no lerping for two frames
        else
          ent->lerpflags |=
              LERP_RESETANIM | LERP_RESETANIM2;  // no lerping for two frames
      }
      // johnfitz
    }
    if (ent->effects & EF_BRIGHTLIGHT) {
      dl = CL_AllocDlight(i);
      VectorCopy(ent->origin, dl->origin);
      dl->origin[2] += 16;
      dl->radius = 400 + (rand() & 31);
      dl->die = CL_Time() + 0.001;
    }
    if (ent->effects & EF_DIMLIGHT) {
      dl = CL_AllocDlight(i);
      VectorCopy(ent->origin, dl->origin);
      dl->radius = 200 + (rand() & 31);
      dl->die = CL_Time() + 0.001;
    }

    if (ent->model->flags & EF_GIB)
      ParticlesAddRocketTrail(oldorg, ent->origin, 2);
    else if (ent->model->flags & EF_ZOMGIB)
      ParticlesAddRocketTrail(oldorg, ent->origin, 4);
    else if (ent->model->flags & EF_TRACER)
      ParticlesAddRocketTrail(oldorg, ent->origin, 3);
    else if (ent->model->flags & EF_TRACER2)
      ParticlesAddRocketTrail(oldorg, ent->origin, 5);
    else if (ent->model->flags & EF_ROCKET) {
      ParticlesAddRocketTrail(oldorg, ent->origin, 0);
      dl = CL_AllocDlight(i);
      VectorCopy(ent->origin, dl->origin);
      dl->radius = 200;
      dl->die = CL_Time() + 0.01;
    } else if (ent->model->flags & EF_GRENADE)
      ParticlesAddRocketTrail(oldorg, ent->origin, 1);
    else if (ent->model->flags & EF_TRACER3)
      ParticlesAddRocketTrail(oldorg, ent->origin, 6);

    ent->forcelink = false;

    if (i == CLViewentityNum() && !Cvar_GetValue(&chase_active)) {
      continue;
    }

    if (cl_numvisedicts < MAX_VISEDICTS) {
      cl_visedicts[cl_numvisedicts] = ent;
      cl_numvisedicts++;
    }
  }
}

/*
=================
CL_Init
=================
*/
void CL_Init(void) {
  CLSMessageClear();

  CL_InitTEnts();

  Cmd_AddCommand("entities", CL_PrintEntities_f);
}

void SetCLWeaponModel(int v) {
  entity_t *view;
  view = &cl_viewent;
  view->model = cl.model_precache[v];
}

void CLPrecacheModelClear(void) {
  memset(cl.model_precache, 0, sizeof(cl.model_precache));
}
