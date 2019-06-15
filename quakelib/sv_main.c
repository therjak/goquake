// sv_main.c -- server main program

#include "_cgo_export.h"
//
#include "quakedef.h"

server_t sv;

static char localmodels[MAX_MODELS][8];  // inline model names for precache

extern qboolean pr_alpha_supported;  // johnfitz

//============================================================================

int ModelLeafIndex(mleaf_t *l) { return l - sv.worldmodel->leafs; }

/*
===============
SV_Init
===============
*/
void SV_Init(void) {
  int i;
  const char *p;
  extern cvar_t sv_gravity;

  sv.edicts = NULL;  // ericw -- sv.edicts switched to use malloc()

  Cvar_FakeRegister(&sv_gravity, "sv_gravity");

  for (i = 0; i < MAX_MODELS; i++) sprintf(localmodels[i], "*%i", i);

  SV_Init_Go();
}

/*
=============================================================================

The PVS must include a small area around the client to allow head bobbing
or other small motion on the client side.  Otherwise, a bob might cause an
entity that should be visible to not show up, especially when the bob
crosses a waterline.

=============================================================================
*/

int fatbytes;
byte fatpvs[MAX_MAP_LEAFS / 8];

// THERJAK
void SV_AddToFatPVS(
    vec3_t org, mnode_t *node,
    qmodel_t *worldmodel)  // johnfitz -- added worldmodel as a parameter
{
  int i;
  byte *pvs;
  mplane_t *plane;
  float d;

  while (1) {
    // if this is a leaf, accumulate the pvs bits
    if (node->contents < 0) {
      if (node->contents != CONTENTS_SOLID) {
        pvs = Mod_LeafPVS((mleaf_t *)node,
                          worldmodel);  // johnfitz -- worldmodel as a parameter
        for (i = 0; i < fatbytes; i++) fatpvs[i] |= pvs[i];
      }
      return;
    }

    plane = node->plane;
    d = DotProduct(org, plane->normal) - plane->dist;
    if (d > 8)
      node = node->children[0];
    else if (d < -8)
      node = node->children[1];
    else {  // go down both
      SV_AddToFatPVS(org, node->children[0],
                     worldmodel);  // johnfitz -- worldmodel as a parameter
      node = node->children[1];
    }
  }
}

/*
=============
SV_FatPVS

Calculates a PVS that is the inclusive or of all leafs within 8 pixels of the
given point.
=============
*/
// THERJAK
byte *SV_FatPVS(
    vec3_t org,
    qmodel_t *worldmodel)  // johnfitz -- added worldmodel as a parameter
{
  fatbytes = (worldmodel->numleafs + 31) >> 3;
  Q_memset(fatpvs, 0, fatbytes);
  SV_AddToFatPVS(org, worldmodel->nodes,
                 worldmodel);  // johnfitz -- worldmodel as a parameter
  return fatpvs;
}

//=============================================================================

/*
=============
SV_WriteEntitiesToClient

=============
*/
void SV_WriteEntitiesToClient(int clent) {
  int e, i;
  int bits;
  byte *pvs;
  vec3_t org;
  float miss;
  int ent;

  // find the client's PVS
  VectorAdd(EVars(clent)->origin, EVars(clent)->view_ofs, org);
  pvs = SV_FatPVS(org, sv.worldmodel);

  // send over all entities (excpet the client) that touch the pvs
  ent = 1;
  for (e = 1; e < SV_NumEdicts(); e++, ent++) {
    if (ent != clent)  // clent is ALLWAYS sent
    {
      // ignore ents without visible models
      if (!EVars(ent)->modelindex || !PR_GetString(EVars(ent)->model)[0])
        continue;

      // johnfitz -- don't send model>255 entities if protocol is 15
      if (SV_Protocol() == PROTOCOL_NETQUAKE &&
          (int)EVars(ent)->modelindex & 0xFF00)
        continue;

      // ignore if not touching a PV leaf
      for (i = 0; i < EDICT_NUM(ent)->num_leafs; i++)
        if (pvs[EDICT_NUM(ent)->leafnums[i] >> 3] &
            (1 << (EDICT_NUM(ent)->leafnums[i] & 7)))
          break;

      // ericw -- added ent->num_leafs < MAX_ENT_LEAFS condition.
      //
      // if ent->num_leafs == MAX_ENT_LEAFS, the ent is visible from too many
      // leafs
      // for us to say whether it's in the PVS, so don't try to vis cull it.
      // this commonly happens with rotators, because they often have huge
      // bboxes
      // spanning the entire map, or really tall lifts, etc.
      if (i == EDICT_NUM(ent)->num_leafs &&
          EDICT_NUM(ent)->num_leafs < MAX_ENT_LEAFS)
        continue;  // not visible
    }

    // johnfitz -- max size for protocol 15 is 18 bytes, not 16 as originally
    // assumed here.  And, for protocol 85 the max size is actually 24 bytes.
    if (SV_MS_Len() + 24 > SV_MS_MaxLen()) {
      // johnfitz -- less spammy overflow message
      if (!dev_overflows.packetsize ||
          dev_overflows.packetsize + CONSOLE_RESPAM_TIME < HostRealTime()) {
        Con_Printf("Packet overflow!\n");
        dev_overflows.packetsize = HostRealTime();
      }
      goto stats;
      // johnfitz
    }

    // send an update
    bits = 0;

    for (i = 0; i < 3; i++) {
      miss = EVars(ent)->origin[i] - EDICT_NUM(ent)->baseline.origin[i];
      if (miss < -0.1 || miss > 0.1) bits |= U_ORIGIN1 << i;
    }

    if (EVars(ent)->angles[0] != EDICT_NUM(ent)->baseline.angles[0])
      bits |= U_ANGLE1;

    if (EVars(ent)->angles[1] != EDICT_NUM(ent)->baseline.angles[1])
      bits |= U_ANGLE2;

    if (EVars(ent)->angles[2] != EDICT_NUM(ent)->baseline.angles[2])
      bits |= U_ANGLE3;

    if (EVars(ent)->movetype == MOVETYPE_STEP)
      bits |= U_STEP;  // don't mess up the step animation

    if (EDICT_NUM(ent)->baseline.colormap != EVars(ent)->colormap)
      bits |= U_COLORMAP;

    if (EDICT_NUM(ent)->baseline.skin != EVars(ent)->skin) bits |= U_SKIN;

    if (EDICT_NUM(ent)->baseline.frame != EVars(ent)->frame) bits |= U_FRAME;

    if (EDICT_NUM(ent)->baseline.effects != EVars(ent)->effects)
      bits |= U_EFFECTS;

    if (EDICT_NUM(ent)->baseline.modelindex != EVars(ent)->modelindex)
      bits |= U_MODEL;

    // johnfitz -- alpha
    if (pr_alpha_supported) {
      // TODO: find a cleaner place to put this code
      UpdateEdictAlpha(ent);
    }

    // don't send invisible entities unless they have effects
    if (EDICT_NUM(ent)->alpha == ENTALPHA_ZERO && !EVars(ent)->effects)
      continue;
    // johnfitz

    // johnfitz -- PROTOCOL_FITZQUAKE
    if (SV_Protocol() != PROTOCOL_NETQUAKE) {
      if (EDICT_NUM(ent)->baseline.alpha != EDICT_NUM(ent)->alpha)
        bits |= U_ALPHA;
      if (bits & U_FRAME && (int)EVars(ent)->frame & 0xFF00) bits |= U_FRAME2;
      if (bits & U_MODEL && (int)EVars(ent)->modelindex & 0xFF00)
        bits |= U_MODEL2;
      if (EDICT_NUM(ent)->sendinterval) bits |= U_LERPFINISH;
      if (bits >= 65536) bits |= U_EXTEND1;
      if (bits >= 16777216) bits |= U_EXTEND2;
    }
    // johnfitz

    if (e >= 256) bits |= U_LONGENTITY;

    if (bits >= 256) bits |= U_MOREBITS;

    //
    // write the message
    //
    SV_MS_WriteByte(bits | U_SIGNAL);

    if (bits & U_MOREBITS) SV_MS_WriteByte(bits >> 8);

    // johnfitz -- PROTOCOL_FITZQUAKE
    if (bits & U_EXTEND1) SV_MS_WriteByte(bits >> 16);
    if (bits & U_EXTEND2) SV_MS_WriteByte(bits >> 24);
    // johnfitz

    if (bits & U_LONGENTITY)
      SV_MS_WriteShort(e);
    else
      SV_MS_WriteByte(e);

    if (bits & U_MODEL) SV_MS_WriteByte(EVars(ent)->modelindex);
    if (bits & U_FRAME) SV_MS_WriteByte(EVars(ent)->frame);
    if (bits & U_COLORMAP) SV_MS_WriteByte(EVars(ent)->colormap);
    if (bits & U_SKIN) SV_MS_WriteByte(EVars(ent)->skin);
    if (bits & U_EFFECTS) SV_MS_WriteByte(EVars(ent)->effects);
    if (bits & U_ORIGIN1) SV_MS_WriteCoord(EVars(ent)->origin[0]);
    if (bits & U_ANGLE1) SV_MS_WriteAngle(EVars(ent)->angles[0]);
    if (bits & U_ORIGIN2) SV_MS_WriteCoord(EVars(ent)->origin[1]);
    if (bits & U_ANGLE2) SV_MS_WriteAngle(EVars(ent)->angles[1]);
    if (bits & U_ORIGIN3) SV_MS_WriteCoord(EVars(ent)->origin[2]);
    if (bits & U_ANGLE3) SV_MS_WriteAngle(EVars(ent)->angles[2]);

    // johnfitz -- PROTOCOL_FITZQUAKE
    if (bits & U_ALPHA) SV_MS_WriteByte(EDICT_NUM(ent)->alpha);
    if (bits & U_FRAME2) SV_MS_WriteByte((int)EVars(ent)->frame >> 8);
    if (bits & U_MODEL2) SV_MS_WriteByte((int)EVars(ent)->modelindex >> 8);
    if (bits & U_LERPFINISH)
      SV_MS_WriteByte(
          (byte)(Q_rint((EVars(ent)->nextthink - SV_Time()) * 255)));
    // johnfitz
  }

// johnfitz -- devstats
stats:
  if (SV_MS_Len() > 1024 && dev_peakstats.packetsize <= 1024)
    Con_DWarning("%i byte packet exceeds standard limit of 1024.\n",
                 SV_MS_Len());
  dev_stats.packetsize = SV_MS_Len();
  dev_peakstats.packetsize = q_max(SV_MS_Len(), dev_peakstats.packetsize);
  // johnfitz
}

/*
==============================================================================

SERVER SPAWNING

==============================================================================
*/

/*
================
SV_SpawnServer

This is called at the start of each level
================
*/
//THERJAK-convert after PR_LoadProgs is no longer needed in c
void SV_SpawnServer(const char *server) {
  static char dummy[8] = {0, 0, 0, 0, 0, 0, 0, 0};
  int ent;
  int i;

  // let's not have any servers with no name
  if (Cvar_GetString(&hostname)[0] == 0) {
    Cvar_Set("hostname", "UNNAMED");
  }

  Con_DPrintf("SpawnServer: %s\n", server);
  SVS_SetChangeLevelIssued(false);  // now safe to issue another

  //
  // tell all connected clients that we are going to a new level
  //
  if (SV_Active()) {
    SV_SendReconnect();
  }

  //
  // make cvars consistant
  //
  if (Cvar_GetValue(&coop)) Cvar_Set("deathmatch", "0");
  current_skill = (int)(Cvar_GetValue(&skill) + 0.5);
  if (current_skill < 0) current_skill = 0;
  if (current_skill > 3) current_skill = 3;

  Cvar_SetValue("skill", (float)current_skill);

  //
  // set up the new server
  //
  // memset (&sv, 0, sizeof(sv));
  Host_ClearMemory();

  SV_SetName(server);

  SV_SetProtocol();  // Go side knows which protocol to set

  if (SV_Protocol() == PROTOCOL_RMQ) {
    // set up the protocol flags used by this server
    // (note - these could be cvar-ised so that server admins could choose the
    // protocol features used by their servers)
    SV_SetProtocolFlags(PRFL_INT32COORD | PRFL_SHORTANGLE);
  } else {
    SV_SetProtocolFlags(0);
  }

  // load progs to get entity field count
  PR_LoadProgsGo();
  PR_LoadProgs();

  // allocate server memory
  /* Host_ClearMemory() called above already cleared the whole sv structure */
  SV_SetMaxEdicts(CLAMP(MIN_EDICTS, (int)Cvar_GetValue(&max_edicts),
                        MAX_EDICTS));  // johnfitz -- max_edicts cvar
  sv.edicts = AllocEdicts();
  // ericw -- sv.edicts switched to use malloc()

  // leave slots at start for clients only
  SV_SetNumEdicts(SVS_GetMaxClients() + 1);
  for (i = 0; i < SV_NumEdicts(); i++) {
    TT_ClearEdict(i);
  }
  // ericw -- sv.edicts switched to use malloc()
  for (i = 0; i < SVS_GetMaxClients(); i++) {
    SetClientEdictId(i, i + 1);
  }

  SV_SetState(ss_loading);
  SV_SetPaused(false);

  SV_SetTime(1.0);

  SV_SetName(server);
  SV_SetModelName("maps/%s.bsp", server);
  sv.worldmodel = Mod_ForName(SV_ModelName(), false);
  if (!sv.worldmodel) {
    Con_Printf("Couldn't spawn server %s\n", SV_ModelName());
    SV_SetActive(false);
    return;
  }
  SVSetWorldModel(sv.worldmodel);
  sv.models[1] = sv.worldmodel;

  //
  // clear world interaction links
  //
  for (i = 1; i < sv.worldmodel->numsubmodels; i++) {
    sv.models[i + 1] = Mod_ForName(localmodels[i], false);
  }

  //
  // load the rest of the entities
  //
  ent = 0;
  TT_ClearEntVars(EVars(ent));
  EDICT_NUM(ent)->free = false;
  EVars(ent)->model = PR_SetEngineString(sv.worldmodel->name);
  EVars(ent)->modelindex = 1;  // world model
  EVars(ent)->solid = SOLID_BSP;
  EVars(ent)->movetype = MOVETYPE_PUSH;

  if (Cvar_GetValue(&coop)) {
    Set_pr_global_struct_coop(Cvar_GetValue(&coop));
  } else {
    Set_pr_global_struct_deathmatch(Cvar_GetValue(&deathmatch));
  }

  Set_pr_global_struct_mapname(PR_SetEngineString(SV_Name()));

  // serverflags are for cross level information (sigils)
  Set_pr_global_struct_serverflags(SVS_GetServerFlags());

  ED_LoadFromFile(sv.worldmodel->entities);

  SV_SetActive(true);

  // all setup is completed, any further precache statements are errors
  SV_SetState(ss_active);

  // run two frames to allow everything to settle
  InitHostFrameTime();
  SV_Physics();
  SV_Physics();

  // create a baseline for more efficient communications
  SV_CreateBaseline();

  // johnfitz -- warn if signon buffer larger than standard server can handle
  if (SV_SO_Len() > 8000 - 2)  // max size that will fit into 8000-sized
                               // client->message buffer with 2 extra
                               // bytes on the end
    Con_DWarning("%i byte signon buffer exceeds standard limit of 7998.\n",
                 SV_SO_Len());
  // johnfitz

  // send serverinfo to all connected clients
  for (i = 0; i < SVS_GetMaxClients(); i++)
    if (GetClientActive(i)) {
      SV_SendServerinfo(i);
    }

  Con_DPrintf("Server spawned.\n");
}

const char *SV_Name() {
  static char buffer[2048];
  char *s = SV_NameInt();
  strncpy(buffer, s, 2048);
  free(s);
  return buffer;
}

const char *SV_ModelName() {
  static char buffer[2048];
  char *s = SV_ModelNameInt();
  strncpy(buffer, s, 2048);
  free(s);
  return buffer;
}
