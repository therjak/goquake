// cl_main.c  -- client main loop

#include "quakedef.h"

// we need to declare some mouse variables here, because the menu system
// references them even when on a unix system.

// these two are not intended to be set directly
cvar_t cl_name;
cvar_t cl_color;
cvar_t cl_shownet;
cvar_t cl_nolerp;
cvar_t cfg_unbindall;
cvar_t lookspring;
cvar_t lookstrafe;
cvar_t sensitivity;
cvar_t m_pitch;
cvar_t cl_forwardspeed;
cvar_t cl_backspeed;
cvar_t cl_movespeedkey;

client_static_t cls;
byte cls_msg_buf[1024];
client_state_t cl;

// FIXME: put these on hunk?
efrag_t cl_efrags[MAX_EFRAGS];
entity_t cl_static_entities[MAX_STATIC_ENTITIES];
lightstyle_t cl_lightstyle[MAX_LIGHTSTYLES];
dlight_t cl_dlights[MAX_DLIGHTS];

entity_t *cl_entities;  // johnfitz -- was a static array, now on hunk
int cl_max_edicts;      // johnfitz -- only changes when new map loads

int cl_numvisedicts;
entity_t *cl_visedicts[MAX_VISEDICTS];

extern cvar_t r_lerpmodels, r_lerpmove;  // johnfitz

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
  memset(cl_lightstyle, 0, sizeof(cl_lightstyle));
  memset(cl_temp_entities, 0, sizeof(cl_temp_entities));
  memset(cl_beams, 0, sizeof(cl_beams));

  // johnfitz -- cl_entities is now dynamically allocated
  cl_max_edicts =
      CLAMP(MIN_EDICTS, (int)Cvar_GetValue(&max_edicts), MAX_EDICTS);
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
=====================
CL_SignonReply

An svc_signonnum has been received, perform a client side setup
=====================
*/
void CL_SignonReply(void) {
  char str[8192];

  Con_DPrintf("CL_SignonReply: %i\n", CLS_GetSignon());

  switch (CLS_GetSignon()) {
    case 1:
      CLSMessageWriteByte(clc_stringcmd);
      CLSMessageWriteString("prespawn");
      break;

    case 2:
      CLSMessageWriteByte(clc_stringcmd);
      CLSMessageWriteString(va("name \"%s\"\n", Cvar_GetString(&cl_name)));

      CLSMessageWriteByte(clc_stringcmd);
      CLSMessageWriteString(va("color %i %i\n",
                               ((int)Cvar_GetValue(&cl_color)) >> 4,
                               ((int)Cvar_GetValue(&cl_color)) & 15));

      CLSMessageWriteByte(clc_stringcmd);
      sprintf(str, "spawn %s", cls.spawnparms);
      CLSMessageWriteString(str);
      break;

    case 3:
      CLSMessageWriteByte(clc_stringcmd);
      CLSMessageWriteString("begin");
      Cache_Report();  // print remaining memory
      break;

    case 4:
      SCR_EndLoadingPlaque();  // allow normal screen updates
      break;
  }
}

/*
=====================
CL_NextDemo

Called to play the next demo in the demo loop
=====================
*/
void CL_NextDemo(void) {
  char str[1024];

  if (CLS_IsDemoCycleStopped()) return;  // don't play demos

  if (!cls.demos[0][0]) {
    Con_Printf("No demos listed with startdemos\n");
    CLS_StopDemoCycle();
    CL_Disconnect();
    return;
  }

  // TODO(therjak): Can this be integrated into CLS_NextDemoInCycle?
  if (!cls.demos[CLS_GetDemoNum()][0] || CLS_GetDemoNum() == MAX_DEMOS) {
    CLS_StartDemoCycle();
  }

  SCR_BeginLoadingPlaque();

  sprintf(str, "playdemo %s\n", cls.demos[CLS_GetDemoNum()]);
  Cbuf_InsertText(str);
  CLS_NextDemoInCycle();
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

  for (i = 0, ent = cl_entities; i < cl.num_entities; i++, ent++) {
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
CL_AllocDlight

===============
*/
dlight_t *CL_AllocDlight(int key) {
  int i;
  dlight_t *dl;

  // first look for an exact key match
  if (key) {
    dl = cl_dlights;
    for (i = 0; i < MAX_DLIGHTS; i++, dl++) {
      if (dl->key == key) {
        memset(dl, 0, sizeof(*dl));
        dl->key = key;
        dl->color[0] = dl->color[1] = dl->color[2] =
            1;  // johnfitz -- lit support via lordhavoc
        return dl;
      }
    }
  }

  // then look for anything else
  dl = cl_dlights;
  for (i = 0; i < MAX_DLIGHTS; i++, dl++) {
    if (dl->die < CL_Time()) {
      memset(dl, 0, sizeof(*dl));
      dl->key = key;
      dl->color[0] = dl->color[1] = dl->color[2] =
          1;  // johnfitz -- lit support via lordhavoc
      return dl;
    }
  }

  dl = &cl_dlights[0];
  memset(dl, 0, sizeof(*dl));
  dl->key = key;
  dl->color[0] = dl->color[1] = dl->color[2] =
      1;  // johnfitz -- lit support via lordhavoc
  return dl;
}

/*
===============
CL_DecayLights

===============
*/
void CL_DecayLights(void) {
  int i;
  dlight_t *dl;
  float time;

  time = CL_Time() - CL_OldTime();

  dl = cl_dlights;
  for (i = 0; i < MAX_DLIGHTS; i++, dl++) {
    if (dl->die < CL_Time() || !dl->radius) continue;

    dl->radius -= time * dl->decay;
    if (dl->radius < 0) dl->radius = 0;
  }
}

/*
===============
CL_LerpPoint

Determines the fraction between the last two messages that the objects
should be put at.
===============
*/
float CL_LerpPoint(void) {
  float f, frac;

  f = CL_MTime() - CL_MTimeOld();

  if (!f || CLS_IsTimeDemo() || SV_Active()) {
    CL_SetTime(CL_MTime());
    return 1;
  }

  if (f > 0.1)  // dropped packet, or start of demo
  {
    CL_SetMTimeOld(CL_MTime() - 0.1);
    f = 0.1;
  }

  frac = (CL_Time() - CL_MTimeOld()) / f;

  if (frac < 0) {
    if (frac < -0.01) CL_SetTime(CL_MTimeOld());
    frac = 0;
  } else if (frac > 1) {
    if (frac > 1.01) CL_SetTime(CL_MTime());
    frac = 1;
  }

  // johnfitz -- better nolerp behavior
  if (Cvar_GetValue(&cl_nolerp)) return 1;
  // johnfitz

  return frac;
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
  for (i = 0; i < 3; i++)
    cl.velocity[i] =
        cl.mvelocity[1][i] + frac * (cl.mvelocity[0][i] - cl.mvelocity[1][i]);

  if (CLS_IsDemoPlayback()) {
    // interpolate the angles
    {
      d = cl.mviewangles[0][0] - cl.mviewangles[1][0];
      if (d > 180)
        d -= 360;
      else if (d < -180)
        d += 360;
      SetCLPitch(cl.mviewangles[1][0] + frac * d);
    }
    {
      d = cl.mviewangles[0][1] - cl.mviewangles[1][1];
      if (d > 180)
        d -= 360;
      else if (d < -180)
        d += 360;
      SetCLYaw(cl.mviewangles[1][1] + frac * d);
    }
    {
      d = cl.mviewangles[0][2] - cl.mviewangles[1][2];
      if (d > 180)
        d -= 360;
      else if (d < -180)
        d += 360;
      SetCLRoll(cl.mviewangles[1][2] + frac * d);
    }
  }

  bobjrotate = anglemod(100 * CL_Time());

  // start on the entity after the world
  for (i = 1, ent = cl_entities + 1; i < cl.num_entities; i++, ent++) {
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

    if (ent->effects & EF_BRIGHTFIELD) R_EntityParticles(ent);

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
        if (ent == &cl_entities[CL_Viewentity()])
          cl.viewent.lerpflags |=
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
      R_RocketTrail(oldorg, ent->origin, 2);
    else if (ent->model->flags & EF_ZOMGIB)
      R_RocketTrail(oldorg, ent->origin, 4);
    else if (ent->model->flags & EF_TRACER)
      R_RocketTrail(oldorg, ent->origin, 3);
    else if (ent->model->flags & EF_TRACER2)
      R_RocketTrail(oldorg, ent->origin, 5);
    else if (ent->model->flags & EF_ROCKET) {
      R_RocketTrail(oldorg, ent->origin, 0);
      dl = CL_AllocDlight(i);
      VectorCopy(ent->origin, dl->origin);
      dl->radius = 200;
      dl->die = CL_Time() + 0.01;
    } else if (ent->model->flags & EF_GRENADE)
      R_RocketTrail(oldorg, ent->origin, 1);
    else if (ent->model->flags & EF_TRACER3)
      R_RocketTrail(oldorg, ent->origin, 6);

    ent->forcelink = false;

    if (i == CL_Viewentity() && !Cvar_GetValue(&chase_active)) continue;

    if (cl_numvisedicts < MAX_VISEDICTS) {
      cl_visedicts[cl_numvisedicts] = ent;
      cl_numvisedicts++;
    }
  }
}

/*
===============
CL_ReadFromServer

Read all incoming data from the server
===============
*/
int CL_ReadFromServer(void) {
  int ret;
  extern int num_temp_entities;  // johnfitz
  int num_beams = 0;             // johnfitz
  int num_dlights = 0;           // johnfitz
  beam_t *b;                     // johnfitz
  dlight_t *l;                   // johnfitz
  int i;                         // johnfitz

  CL_SetOldTime(CL_Time());
  CL_SetTime(CL_Time() + HostFrameTime());

  do {
    ret = CL_GetMessage();
    if (ret == -1) Host_Error("CL_ReadFromServer: lost server connection");
    if (!ret) break;

    CL_SetLastReceivedMessage(HostRealTime());
    CL_ParseServerMessage();
  } while (ret && CLS_GetState() == ca_connected);

  if (Cvar_GetValue(&cl_shownet)) Con_Printf("\n");

  CL_RelinkEntities();
  CL_UpdateTEnts();

  // johnfitz -- devstats

  // visedicts
  if (cl_numvisedicts > 256 && dev_peakstats.visedicts <= 256)
    Con_DWarning("%i visedicts exceeds standard limit of 256.\n",
                 cl_numvisedicts);
  dev_stats.visedicts = cl_numvisedicts;
  dev_peakstats.visedicts = q_max(cl_numvisedicts, dev_peakstats.visedicts);

  // temp entities
  if (num_temp_entities > 64 && dev_peakstats.tempents <= 64)
    Con_DWarning("%i tempentities exceeds standard limit of 64.\n",
                 num_temp_entities);
  dev_stats.tempents = num_temp_entities;
  dev_peakstats.tempents = q_max(num_temp_entities, dev_peakstats.tempents);

  // beams
  for (i = 0, b = cl_beams; i < MAX_BEAMS; i++, b++)
    if (b->model && b->endtime >= CL_Time()) num_beams++;
  if (num_beams > 24 && dev_peakstats.beams <= 24)
    Con_DWarning("%i beams exceeded standard limit of 24.\n", num_beams);
  dev_stats.beams = num_beams;
  dev_peakstats.beams = q_max(num_beams, dev_peakstats.beams);

  // dlights
  for (i = 0, l = cl_dlights; i < MAX_DLIGHTS; i++, l++)
    if (l->die >= CL_Time() && l->radius) num_dlights++;
  if (num_dlights > 32 && dev_peakstats.dlights <= 32)
    Con_DWarning("%i dlights exceeded standard limit of 32.\n", num_dlights);
  dev_stats.dlights = num_dlights;
  dev_peakstats.dlights = q_max(num_dlights, dev_peakstats.dlights);

  // johnfitz

  //
  // bring the links up to date
  //
  return 0;
}

/*
=================
CL_SendCmd
=================
*/
void CL_SendCmd(void) {
  // usercmd_t cmd;

  if (CLS_GetState() != ca_connected) return;

  if (CLS_GetSignon() == SIGNONS) {
    CL_AdjustAngles();
    HandleMove();
  }

  if (CLS_IsDemoPlayback()) {
    CLSMessageClear();
    return;
  }

  // send the reliable message
  if (!CLSHasMessage()) return;  // no message at all

  if (!CLS_NET_CanSendMessage()) {
    Con_DPrintf("CL_SendCmd: can't send\n");
    return;
  }

  if (CLSMessageSend() == -1) Host_Error("CL_SendCmd: lost server connection");

  CLSMessageClear();
}

/*
=============
CL_Tracepos_f -- johnfitz

display impact point of trace along VPN
=============
*/
void CL_Tracepos_f(void) {
  vec3_t v, w;

  if (CLS_GetState() != ca_connected) return;

  VectorMA(r_refdef.vieworg, 8192.0, vpn, v);
  TraceLine(r_refdef.vieworg, v, w);

  if (VectorLength(w) == 0)
    Con_Printf("Tracepos: trace didn't hit anything\n");
  else
    Con_Printf("Tracepos: (%i %i %i)\n", (int)w[0], (int)w[1], (int)w[2]);
}

/*
=============
CL_Viewpos_f -- johnfitz

display client's position and angles
=============
*/
void CL_Viewpos_f(void) {
  if (CLS_GetState() != ca_connected) return;
  // player position
  Con_Printf("Viewpos: (%i %i %i) %i %i %i\n",
             (int)cl_entities[CL_Viewentity()].origin[0],
             (int)cl_entities[CL_Viewentity()].origin[1],
             (int)cl_entities[CL_Viewentity()].origin[2], (int)CLPitch(),
             (int)CLYaw(), (int)CLRoll());
}

/*
=================
CL_Init
=================
*/
void CL_Init(void) {
  CLSMessageClear();

  CL_InitTEnts();

  Cvar_FakeRegister(&cfg_unbindall, "cfg_unbindall");
  Cvar_FakeRegister(&cl_color, "_cl_color");
  Cvar_FakeRegister(&cl_name, "_cl_name");
  Cvar_FakeRegister(&cl_nolerp, "cl_nolerp");
  Cvar_FakeRegister(&cl_shownet, "cl_shownet");

  Cvar_FakeRegister(&cl_backspeed, "cl_backspeed");
  Cvar_FakeRegister(&cl_forwardspeed, "cl_forwardspeed");
  Cvar_FakeRegister(&cl_movespeedkey, "cl_movespeedkey");
  Cvar_FakeRegister(&lookspring, "lookspring");
  Cvar_FakeRegister(&lookstrafe, "lookstrafe");
  Cvar_FakeRegister(&m_pitch, "m_pitch");
  Cvar_FakeRegister(&sensitivity, "sensitivity");

  Cmd_AddCommand("entities", CL_PrintEntities_f);
  Cmd_AddCommand("record", CL_Record_f);
  Cmd_AddCommand("stop", CL_Stop_f);
  Cmd_AddCommand("playdemo", CL_PlayDemo_f);
  Cmd_AddCommand("timedemo", CL_TimeDemo_f);

  Cmd_AddCommand("tracepos", CL_Tracepos_f);
  Cmd_AddCommand("viewpos", CL_Viewpos_f);
}
