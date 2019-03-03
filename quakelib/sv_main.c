// sv_main.c -- server main program

#include "_cgo_export.h"
//
#include "quakedef.h"

server_t sv;

static char localmodels[MAX_MODELS][8];  // inline model names for precache

extern qboolean pr_alpha_supported;  // johnfitz

//============================================================================

/*
===============
SV_Init
===============
*/
void SV_Init(void) {
  int i;
  const char *p;
  extern cvar_t sv_maxvelocity;
  extern cvar_t sv_gravity;
  extern cvar_t sv_nostep;
  extern cvar_t sv_freezenonclients;
  extern cvar_t sv_friction;
  extern cvar_t sv_edgefriction;
  extern cvar_t sv_stopspeed;
  extern cvar_t sv_maxspeed;
  extern cvar_t sv_accelerate;
  extern cvar_t sv_idealpitchscale;
  extern cvar_t sv_aim;
  extern cvar_t sv_altnoclip;

  sv.edicts = NULL;  // ericw -- sv.edicts switched to use malloc()

  Cvar_FakeRegister(&sv_maxvelocity, "sv_maxvelocity");
  Cvar_FakeRegister(&sv_gravity, "sv_gravity");
  Cvar_FakeRegister(&sv_friction, "sv_friction");
  Cvar_SetCallback(&sv_gravity, Host_Callback_Notify);
  Cvar_SetCallback(&sv_friction, Host_Callback_Notify);
  Cvar_FakeRegister(&sv_edgefriction, "edgefriction");
  Cvar_FakeRegister(&sv_stopspeed, "sv_stopspeed");
  Cvar_FakeRegister(&sv_maxspeed, "sv_maxspeed");
  Cvar_SetCallback(&sv_maxspeed, Host_Callback_Notify);
  Cvar_FakeRegister(&sv_accelerate, "sv_accelerate");
  Cvar_FakeRegister(&sv_idealpitchscale, "sv_idealpitchscale");
  Cvar_FakeRegister(&sv_aim, "sv_aim");
  Cvar_FakeRegister(&sv_nostep, "sv_nostep");
  Cvar_FakeRegister(&sv_freezenonclients, "sv_freezenonclients");
  Cvar_FakeRegister(&sv_altnoclip, "sv_altnoclip");

  for (i = 0; i < MAX_MODELS; i++) sprintf(localmodels[i], "*%i", i);

  SV_Init_Go();
}

/*
=============================================================================

EVENT MESSAGES

=============================================================================
*/

/*
==================
SV_StartSound

Each entity can have eight independant sound sources, like voice,
weapon, feet, etc.

Channel 0 is an auto-allocate channel, the others override anything
allready running on that entity/channel pair.

An attenuation of 0 will play full volume everywhere in the level.
Larger attenuations will drop off.  (max 4 attenuation)

==================
*/
void SV_StartSound(int entity, int channel, const char *sample, int volume,
                   float attenuation) {
  int sound_num, ent;
  int i, field_mask;

  if (volume < 0 || volume > 255)
    Host_Error("SV_StartSound: volume = %i", volume);

  if (attenuation < 0 || attenuation > 4)
    Host_Error("SV_StartSound: attenuation = %f", attenuation);

  if (channel < 0 || channel > 7)
    Host_Error("SV_StartSound: channel = %i", channel);

  if (SV_DG_Len() > MAX_DATAGRAM - 16) return;

  // find precache number for sound
  sound_num = ElementOfSVSoundPrecache(sample);

  if (sound_num == -1) {
    Con_Printf("SV_StartSound: %s not precacheed\n", sample);
    return;
  }

  ent = entity;

  field_mask = 0;
  if (volume != DEFAULT_SOUND_PACKET_VOLUME) field_mask |= SND_VOLUME;
  if (attenuation != DEFAULT_SOUND_PACKET_ATTENUATION)
    field_mask |= SND_ATTENUATION;

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (ent >= 8192) {
    if (SV_Protocol() == PROTOCOL_NETQUAKE)
      return;  // don't send any info protocol can't support
    else
      field_mask |= SND_LARGEENTITY;
  }
  if (sound_num >= 256 || channel >= 8) {
    if (SV_Protocol() == PROTOCOL_NETQUAKE)
      return;  // don't send any info protocol can't support
    else
      field_mask |= SND_LARGESOUND;
  }
  // johnfitz

  // directed messages go only to the entity the are targeted on
  SV_DG_WriteByte(svc_sound);
  SV_DG_WriteByte(field_mask);
  if (field_mask & SND_VOLUME) SV_DG_WriteByte(volume);
  if (field_mask & SND_ATTENUATION) SV_DG_WriteByte(attenuation * 64);

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (field_mask & SND_LARGEENTITY) {
    SV_DG_WriteShort(ent);
    SV_DG_WriteByte(channel);
  } else
    SV_DG_WriteShort((ent << 3) | channel);
  if (field_mask & SND_LARGESOUND)
    SV_DG_WriteShort(sound_num);
  else
    SV_DG_WriteByte(sound_num);
  // johnfitz

  for (i = 0; i < 3; i++)
    SV_DG_WriteCoord(EVars(entity)->origin[i] +
                     0.5 * (EVars(entity)->mins[i] + EVars(entity)->maxs[i]));
}

/*
==============================================================================

CLIENT SPAWNING

==============================================================================
*/

/*
================
SV_ConnectClient

Initializes a client_t for a new net connection.  This will only be called
once for a player each game, not once for each level change.
================

*/
void SV_ConnectClient(int clientnum) {
  int client;
  int edictnum;
  int i;
  float spawn_parms[NUM_SPAWN_PARMS];

  client = clientnum;

  Con_DPrintf("Client %s connected\n", NET_QSocketGetAddressString(client));

  edictnum = clientnum + 1;

  // set up the client_t
  if (SV_LoadGame()) {
    for (i = 0; i < NUM_SPAWN_PARMS; i++) {
      spawn_parms[i] = GetClientSpawnParam(client, i);
    }
  }
  CleanSVClient(client);

  SetClientName(client, "unconnected");
  SetClientActive(client, true);
  SetClientSpawned(client, false);
  SetClientEdictId(client, edictnum);

  if (SV_LoadGame()) {
    for (i = 0; i < NUM_SPAWN_PARMS; i++) {
      SetClientSpawnParam(client, i, spawn_parms[i]);
    }
  } else {
    // call the progs to get default spawn parms for the new client
    PR_ExecuteProgram(Pr_global_struct_SetNewParms());
    for (i = 0; i < NUM_SPAWN_PARMS; i++)
      SetClientSpawnParam(client, i, Pr_global_struct_parm(i));
  }
  SV_SendServerinfo(client);
}

/*
===============================================================================

FRAME UPDATES

===============================================================================
*/

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
        if (pvs[EDICT_NUM(ent)->leafnums[i] >> 3] & (1 << (EDICT_NUM(ent)->leafnums[i] & 7))) break;

      // ericw -- added ent->num_leafs < MAX_ENT_LEAFS condition.
      //
      // if ent->num_leafs == MAX_ENT_LEAFS, the ent is visible from too many
      // leafs
      // for us to say whether it's in the PVS, so don't try to vis cull it.
      // this commonly happens with rotators, because they often have huge
      // bboxes
      // spanning the entire map, or really tall lifts, etc.
      if (i == EDICT_NUM(ent)->num_leafs && EDICT_NUM(ent)->num_leafs < MAX_ENT_LEAFS)
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

    if (EVars(ent)->angles[0] != EDICT_NUM(ent)->baseline.angles[0]) bits |= U_ANGLE1;

    if (EVars(ent)->angles[1] != EDICT_NUM(ent)->baseline.angles[1]) bits |= U_ANGLE2;

    if (EVars(ent)->angles[2] != EDICT_NUM(ent)->baseline.angles[2]) bits |= U_ANGLE3;

    if (EVars(ent)->movetype == MOVETYPE_STEP)
      bits |= U_STEP;  // don't mess up the step animation

    if (EDICT_NUM(ent)->baseline.colormap != EVars(ent)->colormap) bits |= U_COLORMAP;

    if (EDICT_NUM(ent)->baseline.skin != EVars(ent)->skin) bits |= U_SKIN;

    if (EDICT_NUM(ent)->baseline.frame != EVars(ent)->frame) bits |= U_FRAME;

    if (EDICT_NUM(ent)->baseline.effects != EVars(ent)->effects) bits |= U_EFFECTS;

    if (EDICT_NUM(ent)->baseline.modelindex != EVars(ent)->modelindex) bits |= U_MODEL;

    // johnfitz -- alpha
    if (pr_alpha_supported) {
      // TODO: find a cleaner place to put this code
      eval_t *val;
      val = GetEdictFieldValue(EVars(ent), "alpha");
      if (val) EDICT_NUM(ent)->alpha = ENTALPHA_ENCODE(val->_float);
    }

    // don't send invisible entities unless they have effects
    if (EDICT_NUM(ent)->alpha == ENTALPHA_ZERO && !EVars(ent)->effects) continue;
    // johnfitz

    // johnfitz -- PROTOCOL_FITZQUAKE
    if (SV_Protocol() != PROTOCOL_NETQUAKE) {
      if (EDICT_NUM(ent)->baseline.alpha != EDICT_NUM(ent)->alpha) bits |= U_ALPHA;
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
=============
SV_CleanupEnts

=============
*/
void SV_CleanupEnts(void) {
  int e;
  int ent;

  ent = 1;
  for (e = 1; e < SV_NumEdicts(); e++, ent++) {
    EVars(ent)->effects = (int)EVars(ent)->effects & ~EF_MUZZLEFLASH;
  }
}

/*
==================
SV_WriteClientdataToMessage

==================
*/
void SV_WriteClientdataToMessage(int ent) {
  int bits;
  int i;
  int other;
  int items;
  eval_t *val;

  //
  // send a damage message
  //
  if (EVars(ent)->dmg_take || EVars(ent)->dmg_save) {
    other = EVars(ent)->dmg_inflictor;
    SV_MS_WriteByte(svc_damage);
    SV_MS_WriteByte(EVars(ent)->dmg_save);
    SV_MS_WriteByte(EVars(ent)->dmg_take);
    for (i = 0; i < 3; i++)
      SV_MS_WriteCoord(EVars(other)->origin[i] +
                       0.5 * (EVars(other)->mins[i] + EVars(other)->maxs[i]));

    EVars(ent)->dmg_take = 0;
    EVars(ent)->dmg_save = 0;
  }

  //
  // send the current viewpos offset from the view entity
  //
  SV_SetIdealPitch();  // how much to look up / down ideally

  // a fixangle might get lost in a dropped packet.  Oh well.
  if (EVars(ent)->fixangle) {
    SV_MS_WriteByte(svc_setangle);
    for (i = 0; i < 3; i++) SV_MS_WriteAngle(EVars(ent)->angles[i]);
    EVars(ent)->fixangle = 0;
  }

  bits = 0;

  if (EVars(ent)->view_ofs[2] != DEFAULT_VIEWHEIGHT) bits |= SU_VIEWHEIGHT;

  if (EVars(ent)->idealpitch) bits |= SU_IDEALPITCH;

  // stuff the sigil bits into the high bits of items for sbar, or else
  // mix in items2
  val = GetEdictFieldValue(EVars(ent), "items2");

  if (val)
    items = (int)EVars(ent)->items | ((int)val->_float << 23);
  else
    items =
        (int)EVars(ent)->items | ((int)Pr_global_struct_serverflags() << 28);

  bits |= SU_ITEMS;

  if ((int)EVars(ent)->flags & FL_ONGROUND) bits |= SU_ONGROUND;

  if (EVars(ent)->waterlevel >= 2) bits |= SU_INWATER;

  for (i = 0; i < 3; i++) {
    if (EVars(ent)->punchangle[i]) bits |= (SU_PUNCH1 << i);
    if (EVars(ent)->velocity[i]) bits |= (SU_VELOCITY1 << i);
  }

  if (EVars(ent)->weaponframe) bits |= SU_WEAPONFRAME;

  if (EVars(ent)->armorvalue) bits |= SU_ARMOR;

  //	if (ent->v.weapon)
  bits |= SU_WEAPON;

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (SV_Protocol() != PROTOCOL_NETQUAKE) {
    if (bits & SU_WEAPON &&
        SV_ModelIndex(PR_GetString(EVars(ent)->weaponmodel)) & 0xFF00)
      bits |= SU_WEAPON2;
    if ((int)EVars(ent)->armorvalue & 0xFF00) bits |= SU_ARMOR2;
    if ((int)EVars(ent)->currentammo & 0xFF00) bits |= SU_AMMO2;
    if ((int)EVars(ent)->ammo_shells & 0xFF00) bits |= SU_SHELLS2;
    if ((int)EVars(ent)->ammo_nails & 0xFF00) bits |= SU_NAILS2;
    if ((int)EVars(ent)->ammo_rockets & 0xFF00) bits |= SU_ROCKETS2;
    if ((int)EVars(ent)->ammo_cells & 0xFF00) bits |= SU_CELLS2;
    if (bits & SU_WEAPONFRAME && (int)EVars(ent)->weaponframe & 0xFF00)
      bits |= SU_WEAPONFRAME2;
    if (bits & SU_WEAPON && EDICT_NUM(ent)->alpha != ENTALPHA_DEFAULT)
      bits |= SU_WEAPONALPHA;  // for now, weaponalpha = client entity alpha
    if (bits >= 65536) bits |= SU_EXTEND1;
    if (bits >= 16777216) bits |= SU_EXTEND2;
  }
  // johnfitz

  // send the data

  SV_MS_WriteByte(svc_clientdata);
  SV_MS_WriteShort(bits);

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (bits & SU_EXTEND1) SV_MS_WriteByte(bits >> 16);
  if (bits & SU_EXTEND2) SV_MS_WriteByte(bits >> 24);
  // johnfitz

  if (bits & SU_VIEWHEIGHT) SV_MS_WriteChar(EVars(ent)->view_ofs[2]);

  if (bits & SU_IDEALPITCH) SV_MS_WriteChar(EVars(ent)->idealpitch);

  for (i = 0; i < 3; i++) {
    if (bits & (SU_PUNCH1 << i)) SV_MS_WriteChar(EVars(ent)->punchangle[i]);
    if (bits & (SU_VELOCITY1 << i))
      SV_MS_WriteChar(EVars(ent)->velocity[i] / 16);
  }

  // [always sent]	if (bits & SU_ITEMS)
  SV_MS_WriteLong(items);

  if (bits & SU_WEAPONFRAME) SV_MS_WriteByte(EVars(ent)->weaponframe);
  if (bits & SU_ARMOR) SV_MS_WriteByte(EVars(ent)->armorvalue);
  if (bits & SU_WEAPON)
    SV_MS_WriteByte(SV_ModelIndex(PR_GetString(EVars(ent)->weaponmodel)));

  SV_MS_WriteShort(EVars(ent)->health);
  SV_MS_WriteByte(EVars(ent)->currentammo);
  SV_MS_WriteByte(EVars(ent)->ammo_shells);
  SV_MS_WriteByte(EVars(ent)->ammo_nails);
  SV_MS_WriteByte(EVars(ent)->ammo_rockets);
  SV_MS_WriteByte(EVars(ent)->ammo_cells);

  if (CMLStandardQuake()) {
    SV_MS_WriteByte(EVars(ent)->weapon);
  } else {
    for (i = 0; i < 32; i++) {
      if (((int)EVars(ent)->weapon) & (1 << i)) {
        SV_MS_WriteByte(i);
        break;
      }
    }
  }

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (bits & SU_WEAPON2)
    SV_MS_WriteByte(SV_ModelIndex(PR_GetString(EVars(ent)->weaponmodel)) >> 8);
  if (bits & SU_ARMOR2) SV_MS_WriteByte((int)EVars(ent)->armorvalue >> 8);
  if (bits & SU_AMMO2) SV_MS_WriteByte((int)EVars(ent)->currentammo >> 8);
  if (bits & SU_SHELLS2) SV_MS_WriteByte((int)EVars(ent)->ammo_shells >> 8);
  if (bits & SU_NAILS2) SV_MS_WriteByte((int)EVars(ent)->ammo_nails >> 8);
  if (bits & SU_ROCKETS2) SV_MS_WriteByte((int)EVars(ent)->ammo_rockets >> 8);
  if (bits & SU_CELLS2) SV_MS_WriteByte((int)EVars(ent)->ammo_cells >> 8);
  if (bits & SU_WEAPONFRAME2)
    SV_MS_WriteByte((int)EVars(ent)->weaponframe >> 8);
  // for now, weaponalpha = client entity alpha
  if (bits & SU_WEAPONALPHA) SV_MS_WriteByte(EDICT_NUM(ent)->alpha);
  // johnfitz
}

/*
=======================
SV_SendClientDatagram
=======================
*/
qboolean SV_SendClientDatagram(int client) {
  SV_MS_Clear();
  SV_MS_SetMaxLen(MAX_DATAGRAM);

  // johnfitz -- if client is nonlocal, use smaller max size so packets aren't
  // fragmented
  if (Q_strcmp(NET_QSocketGetAddressString(client), "LOCAL") != 0)
    SV_MS_SetMaxLen(DATAGRAM_MTU);
  // johnfitz

  SV_MS_WriteByte(svc_time);
  SV_MS_WriteFloat(SV_Time());

  // add the client specific data to the datagram
  SV_WriteClientdataToMessage(GetClientEdictId(client));

  SV_WriteEntitiesToClient(GetClientEdictId(client));

  return SV_DG_SendOut(client);
}

/*
=======================
SV_UpdateToReliableMessages
=======================
*/
void SV_UpdateToReliableMessages(void) {
  int i, j;

  // check for changes to be sent over the reliable streams
  for (i = 0; i < SVS_GetMaxClients(); i++) {
    if (GetClientOldFrags(i) != EVars(GetClientEdictId(i))->frags) {
      for (j = 0; j < SVS_GetMaxClients(); j++) {
        if (!GetClientActive(j)) continue;
        ClientWriteByte(j, svc_updatefrags);
        ClientWriteByte(j, i);
        ClientWriteShort(j, EVars(GetClientEdictId(i))->frags);
      }

      SetClientOldFrags(i, EVars(GetClientEdictId(i))->frags);
    }
  }

  SV_RD_SendOut();
}

/*
=======================
SV_SendClientMessages
=======================
*/
void SV_SendClientMessages(void) {
  int i;

  // update frags, names, etc
  SV_UpdateToReliableMessages();

  // build individual updates
  for (i = 0; i < SVS_GetMaxClients(); i++) {
    if (!GetClientActive(i)) continue;

    if (GetClientSpawned(i)) {
      if (!SV_SendClientDatagram(i)) continue;
    } else {
      // the player isn't totally in the game yet
      // send small keepalive messages if too much time has passed
      // send a full message when the next signon stage has been requested
      // some other message data (name changes, etc) may accumulate
      // between signon stages
      if (!GetClientSendSignon(i)) {
        if (HostRealTime() - GetClientLastMessage(i) > 5) SV_SendNop(i);
        continue;  // don't send out non-signon messages
      }
    }

    // check for an overflowed message.  Should only happen
    // on a very fucked up connection that backs up a lot, then
    // changes level
    if (GetClientOverflowed(i)) {
      SV_DropClient(i, true);
      SetClientOverflowed(i, false);
      continue;
    }

    if (ClientHasMessage(i)) {
      if (!ClientCanSendMessage(i)) {
        //				I_Printf ("can't write\n");
        continue;
      }

      if (ClientSendMessage(i) == -1) {
        // if the message couldn't send, kick off
        SV_DropClient(i, true);
      }
      ClientClearMessage(i);
      SetClientLastMessage(i);
      SetClientSendSignon(i, false);
    }
  }

  // clear muzzle flashes
  SV_CleanupEnts();
}

/*
==============================================================================

SERVER SPAWNING

==============================================================================
*/

/*
================
SV_CreateBaseline
================
*/
void SV_CreateBaseline(void) {
  int i;
  edict_t *svent;
  int entnum;
  int bits;  // johnfitz -- PROTOCOL_FITZQUAKE

  for (entnum = 0; entnum < SV_NumEdicts(); entnum++) {
    // get the current server version
    svent = EDICT_NUM(entnum);
    if (svent->free) continue;
    if (entnum > SVS_GetMaxClients() && !EdictV(svent)->modelindex) continue;

    //
    // create entity baseline
    //
    VectorCopy(EdictV(svent)->origin, svent->baseline.origin);
    VectorCopy(EdictV(svent)->angles, svent->baseline.angles);
    svent->baseline.frame = EdictV(svent)->frame;
    svent->baseline.skin = EdictV(svent)->skin;
    if (entnum > 0 && entnum <= SVS_GetMaxClients()) {
      svent->baseline.colormap = entnum;
      svent->baseline.modelindex = SV_ModelIndex("progs/player.mdl");
      svent->baseline.alpha = ENTALPHA_DEFAULT;  // johnfitz -- alpha support
    } else {
      svent->baseline.colormap = 0;
      svent->baseline.modelindex =
          SV_ModelIndex(PR_GetString(EdictV(svent)->model));
      svent->baseline.alpha = svent->alpha;  // johnfitz -- alpha support
    }

    // johnfitz -- PROTOCOL_FITZQUAKE
    bits = 0;
    if (SV_Protocol() == PROTOCOL_NETQUAKE)  // still want to send baseline in
                                             // PROTOCOL_NETQUAKE, so reset
                                             // these values
    {
      if (svent->baseline.modelindex & 0xFF00) svent->baseline.modelindex = 0;
      if (svent->baseline.frame & 0xFF00) svent->baseline.frame = 0;
      svent->baseline.alpha = ENTALPHA_DEFAULT;
    } else  // decide which extra data needs to be sent
    {
      if (svent->baseline.modelindex & 0xFF00) bits |= B_LARGEMODEL;
      if (svent->baseline.frame & 0xFF00) bits |= B_LARGEFRAME;
      if (svent->baseline.alpha != ENTALPHA_DEFAULT) bits |= B_ALPHA;
    }
    // johnfitz

    //
    // add to the message
    //
    // johnfitz -- PROTOCOL_FITZQUAKE
    if (bits)
      SV_SO_WriteByte(svc_spawnbaseline2);
    else
      SV_SO_WriteByte(svc_spawnbaseline);
    // johnfitz

    SV_SO_WriteShort(entnum);

    // johnfitz -- PROTOCOL_FITZQUAKE
    if (bits) SV_SO_WriteByte(bits);

    if (bits & B_LARGEMODEL)
      SV_SO_WriteShort(svent->baseline.modelindex);
    else
      SV_SO_WriteByte(svent->baseline.modelindex);

    if (bits & B_LARGEFRAME)
      SV_SO_WriteShort(svent->baseline.frame);
    else
      SV_SO_WriteByte(svent->baseline.frame);
    // johnfitz

    SV_SO_WriteByte(svent->baseline.colormap);
    SV_SO_WriteByte(svent->baseline.skin);
    for (i = 0; i < 3; i++) {
      SV_SO_WriteCoord(svent->baseline.origin[i]);
      SV_SO_WriteAngle(svent->baseline.angles[i]);
    }

    // johnfitz -- PROTOCOL_FITZQUAKE
    if (bits & B_ALPHA) SV_SO_WriteByte(svent->baseline.alpha);
    // johnfitz
  }
}

/*
================
SV_SaveSpawnparms

Grabs the current state of each client for saving across the
transition to another level
================
*/
void SV_SaveSpawnparms(void) {
  int i, j;

  SVS_SetServerFlags(Pr_global_struct_serverflags());

  for (i = 0, host_client = 0; i < SVS_GetMaxClients(); i++, host_client++) {
    if (!GetClientActive(HostClient())) continue;

    // call the progs to get default spawn parms for the new client
    Set_pr_global_struct_self(GetClientEdictId(HostClient()));
    PR_ExecuteProgram(Pr_global_struct_SetChangeParms());
    for (j = 0; j < NUM_SPAWN_PARMS; j++) {
      SetClientSpawnParam(HostClient(), j, Pr_global_struct_parm(j));
    }
  }
}

/*
================
SV_SpawnServer

This is called at the start of each level
================
*/
extern float scr_centertime_off;
void SV_SpawnServer(const char *server) {
  static char dummy[8] = {0, 0, 0, 0, 0, 0, 0, 0};
  edict_t *ent;
  int i;

  // let's not have any servers with no name
  if (Cvar_GetString(&hostname)[0] == 0) {
    Cvar_Set("hostname", "UNNAMED");
  }
  scr_centertime_off = 0;

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
  PR_LoadProgs();
  PR_LoadProgsGo();

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
  sv.models[1] = sv.worldmodel;

  //
  // clear world interaction links
  //
  SV_ClearWorld();

  SetSVSoundPrecache(0, dummy);
  SetSVModelPrecache(0, dummy);
  SetSVModelPrecache(1, SV_ModelName());
  for (i = 1; i < sv.worldmodel->numsubmodels; i++) {
    SetSVModelPrecache(1 + i, localmodels[i]);
    sv.models[i + 1] = Mod_ForName(localmodels[i], false);
  }

  //
  // load the rest of the entities
  //
  ent = EDICT_NUM(0);
  TT_ClearEntVars(EdictV(ent));
  ent->free = false;
  EdictV(ent)->model = PR_SetEngineString(sv.worldmodel->name);
  EdictV(ent)->modelindex = 1;  // world model
  EdictV(ent)->solid = SOLID_BSP;
  EdictV(ent)->movetype = MOVETYPE_PUSH;

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
