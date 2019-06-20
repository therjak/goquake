// sv_main.c -- server main program

#include "_cgo_export.h"
//
#include "quakedef.h"

server_t sv;

static char localmodels[MAX_MODELS][8];  // inline model names for precache

extern qboolean pr_alpha_supported;  // johnfitz

//============================================================================

cvar_t sv_gravity;
/*
===============
SV_Init
===============
*/
void SV_Init(void) {
  int i;
  const char *p;

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

void SV_AddToFatPVS(vec3_t org, mnode_t *node, qmodel_t *worldmodel)
{
  int i;
  byte *pvs;
  mplane_t *plane;
  float d;

  while (1) {
    // if this is a leaf, accumulate the pvs bits
    if (node->contents < 0) {
      if (node->contents != CONTENTS_SOLID) {
        pvs = Mod_LeafPVS((mleaf_t *)node, worldmodel);  
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
      SV_AddToFatPVS(org, node->children[0], worldmodel);
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
byte *SV_FatPVS(
    vec3_t org,
    qmodel_t *worldmodel)
{
  fatbytes = (worldmodel->numleafs + 31) >> 3;
  Q_memset(fatpvs, 0, fatbytes);
  SV_AddToFatPVS(org, worldmodel->nodes, worldmodel);
  return fatpvs;
}

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
  AllocEdicts();
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
  LoadModelGo(SV_ModelName());
  sv.worldmodel = Mod_ForName(SV_ModelName(), false);
  if (!sv.worldmodel) {
    Con_Printf("Couldn't spawn server %s\n", SV_ModelName());
    SV_SetActive(false);
    return;
  }
  SVSetWorldModelByName(SV_ModelName());

  // load the rest of the entities
  ent = 0;
  TT_ClearEntVars(EVars(ent));
  EDICT_SETFREE(ent, false);
  EVars(ent)->model = PR_SetEngineString(SV_ModelName());
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
