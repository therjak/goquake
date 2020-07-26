#include "_cgo_export.h"
#include "quakedef.h"

#include "dlight.h"


const char* BOLT1 = "progs/bolt.mdl";
const char* BOLT2 = "progs/bolt2.mdl";
const char* BOLT3 = "progs/bolt3.mdl";
const char* BEAM = "progs/beam.mdl";

const char *CL_MSG_ReadString(void) {
  static char string[2048];
  int c;
  size_t l;

  l = 0;
  do {
    c = CL_MSG_ReadByte();
    if (c == -1 || c == 0) break;
    string[l] = c;
    l++;
  } while (l < sizeof(string) - 1);

  string[l] = 0;
  return string;
}

int num_temp_entities;
entity_t cl_temp_entities[MAX_TEMP_ENTITIES];
beam_t cl_beams[MAX_BEAMS];

void CL_ParseBeam(const char *name, int ent, float s1, float s2, float s3,
                  float e1, float e2, float e3) {
  // int ent;
  vec3_t start, end;
  beam_t *b;
  int i;
  qmodel_t *m;
  m = Mod_ForName(name, true);

  start[0] = s1;
  start[1] = s2;
  start[2] = s3;

  end[0] = e1;
  end[1] = e2;
  end[2] = e3;

  // override any beam with the same entity
  for (i = 0, b = cl_beams; i < MAX_BEAMS; i++, b++)
    if (b->entity == ent) {
      b->entity = ent;
      b->model = m;
      b->endtime = CL_Time() + 0.2;
      VectorCopy(start, b->start);
      VectorCopy(end, b->end);
      return;
    }

  // find a free beam
  for (i = 0, b = cl_beams; i < MAX_BEAMS; i++, b++) {
    if (!b->model || b->endtime < CL_Time()) {
      b->entity = ent;
      b->model = m;
      b->endtime = CL_Time() + 0.2;
      VectorCopy(start, b->start);
      VectorCopy(end, b->end);
      return;
    }
  }

  // johnfitz -- less spammy overflow message
  if (!dev_overflows.beams ||
      dev_overflows.beams + CONSOLE_RESPAM_TIME < HostRealTime()) {
    Con_Printf("Beam list overflow!\n");
    dev_overflows.beams = HostRealTime();
  }
  // johnfitz
}
/*
=================
CL_NewTempEntity
=================
*/
entity_t *CL_NewTempEntity(void) {
  entity_t *ent;

  if (cl_numvisedicts == MAX_VISEDICTS) return NULL;
  if (num_temp_entities == MAX_TEMP_ENTITIES) return NULL;
  ent = &cl_temp_entities[num_temp_entities];
  memset(ent, 0, sizeof(*ent));
  num_temp_entities++;
  cl_visedicts[cl_numvisedicts] = ent;
  cl_numvisedicts++;

  return ent;
}

/*
=================
CL_UpdateTEnts
=================
*/
void CL_UpdateTEnts(void) {
  int i, j;  // johnfitz -- use j instead of using i twice, so we don't corrupt
             // memory
  beam_t *b;
  vec3_t dist, org;
  float d;
  entity_t *ent;
  float yaw, pitch;
  float forward;

  num_temp_entities = 0;

  srand((int)(CL_Time() * 1000));  // johnfitz -- freeze beams when paused

  // update lightning
  for (i = 0, b = cl_beams; i < MAX_BEAMS; i++, b++) {
    if (!b->model || b->endtime < CL_Time()) continue;

    // if coming from the player, update the start position
    if (b->entity == CL_Viewentity()) {
      VectorCopy(cl_entities[CL_Viewentity()].origin, b->start);
    }

    // calculate pitch and yaw
    VectorSubtract(b->end, b->start, dist);

    if (dist[1] == 0 && dist[0] == 0) {
      yaw = 0;
      if (dist[2] > 0)
        pitch = 90;
      else
        pitch = 270;
    } else {
      yaw = (int)(atan2(dist[1], dist[0]) * 180 / M_PI);
      if (yaw < 0) yaw += 360;

      forward = sqrt(dist[0] * dist[0] + dist[1] * dist[1]);
      pitch = (int)(atan2(dist[2], forward) * 180 / M_PI);
      if (pitch < 0) pitch += 360;
    }

    // add new entities for the lightning
    VectorCopy(b->start, org);
    d = VectorNormalize(dist);
    while (d > 0) {
      ent = CL_NewTempEntity();
      if (!ent) return;
      VectorCopy(org, ent->origin);
      ent->model = b->model;
      ent->angles[0] = pitch;
      ent->angles[1] = yaw;
      ent->angles[2] = rand() % 360;

      // johnfitz -- use j instead of using i twice, so we don't corrupt memory
      for (j = 0; j < 3; j++) org[j] += dist[j] * 30;
      d -= 30;
    }
  }
}

qboolean warn_about_nehahra_protocol;  // johnfitz

/*
==================
CL_ParseServerInfo
==================
*/
void CL_ParseServerInfo(void) {
  const char *str;
  int i;
  int nummodels, numsounds;
  char model_precache[MAX_MODELS][MAX_QPATH];
  char sound_precache[MAX_SOUNDS][MAX_QPATH];

  Con_DPrintf("Serverinfo packet received.\n");

  // ericw -- bring up loading plaque for map changes within a demo.
  //          it will be hidden in CL_SignonReply.
  if (CLS_IsDemoPlayback()) SCR_BeginLoadingPlaque();

  //
  // wipe the client_state_t struct
  //
  CL_ClearState();

  // parse protocol version number
  i = CL_MSG_ReadLong();
  // johnfitz -- support multiple protocols
  if (i != PROTOCOL_NETQUAKE && i != PROTOCOL_FITZQUAKE && i != PROTOCOL_RMQ) {
    Con_Printf("\n");  // because there's no newline after serverinfo print
    Host_Error("Server returned version %i, not %i or %i or %i", i,
               PROTOCOL_NETQUAKE, PROTOCOL_FITZQUAKE, PROTOCOL_RMQ);
  }
  CL_SetProtocol(i);
  // johnfitz

  if (CL_Protocol() == PROTOCOL_RMQ) {
    const unsigned int supportedflags =
        (PRFL_SHORTANGLE | PRFL_FLOATANGLE | PRFL_24BITCOORD | PRFL_FLOATCOORD |
         PRFL_EDICTSCALE | PRFL_INT32COORD);

    // mh - read protocol flags from server so that we know what protocol
    // features to expect
    CL_SetProtocolFlags((unsigned int)CL_MSG_ReadLong());

    if (0 != (CL_ProtocolFlags() & (~supportedflags))) {
      Con_Warning("PROTOCOL_RMQ protocolflags %i contains unsupported flags\n",
                  CL_ProtocolFlags());
    }
  } else
    CL_SetProtocolFlags(0);

  // parse maxclients
  CL_SetMaxClients(CL_MSG_ReadByte());
  if (CL_MaxClients() < 1 || CL_MaxClients() > MAX_SCOREBOARD) {
    Host_Error("Bad maxclients (%u) from server", CL_MaxClients());
  }

  // parse gametype
  CL_SetGameType(CL_MSG_ReadByte());

  // parse signon message
  str = CL_MSG_ReadString();
  CL_SetLevelName(str);

  // seperate the printfs so the server message can have a color
  ConPrintBar();
  Con_Printf("%c%s\n", 2, str);

  // johnfitz -- tell user which protocol this is
  Con_Printf("Using protocol %i\n", i);

  // first we go through and touch all of the precache data that still
  // happens to be in the cache, so precaching something else doesn't
  // needlessly purge it

  // precache models
  memset(cl.model_precache, 0, sizeof(cl.model_precache));
  for (nummodels = 1;; nummodels++) {
    str = CL_MSG_ReadString();
    if (!str[0]) break;
    if (nummodels == MAX_MODELS) {
      Host_Error("Server sent too many model precaches");
    }
    q_strlcpy(model_precache[nummodels], str, MAX_QPATH);
    Mod_TouchModel(str);
  }

  // johnfitz -- check for excessive models
  if (nummodels >= 256)
    Con_DWarning("%i models exceeds standard limit of 256.\n", nummodels);
  // johnfitz

  // precache sounds
  CL_SoundPrecacheClear();
  for (numsounds = 1;; numsounds++) {
    str = CL_MSG_ReadString();
    if (!str[0]) break;
    if (numsounds == MAX_SOUNDS) {
      Host_Error("Server sent too many sound precaches");
    }
    q_strlcpy(sound_precache[numsounds], str, MAX_QPATH);
    S_TouchSound(str);
  }

  // johnfitz -- check for excessive sounds
  if (numsounds >= 256)
    Con_DWarning("%i sounds exceeds standard limit of 256.\n", numsounds);
  // johnfitz

  //
  // now we try to load everything else until a cache allocation fails
  //

  // copy the naked name of the map file to the cl structure -- O.S
  char mapname[128];
  COM_StripExtension(COM_SkipPath(model_precache[1]), mapname,
                     sizeof(mapname));
  CL_SetMapName(mapname);

  for (i = 1; i < nummodels; i++) {
    cl.model_precache[i] = Mod_ForName(model_precache[i], false);
    if (cl.model_precache[i] == NULL) {
      Host_Error("Model %s not found", model_precache[i]);
    }
    CL_KeepaliveMessage();
  }

  for (i = 1; i < numsounds; i++) {
    int s = S_PrecacheSound(sound_precache[i]);
    CL_SoundPrecacheAdd(s);

    CL_KeepaliveMessage();
  }

  // local state
  cl_entities[0].model = cl.worldmodel = cl.model_precache[1];
  CLSetWorldModel(cl.worldmodel);  // notify the go side

  R_NewMap();

  // johnfitz -- clear out string; we don't consider identical
  // messages to be duplicates if the map has changed in between
  Con_ResetLastCenterString();
  // johnfitz

  Hunk_Check();  // make sure nothing is hurt

  noclip_anglehack = false;  // noclip is turned off at start

  warn_about_nehahra_protocol = true;  // johnfitz -- warn about nehahra
                                       // protocol hack once per server
                                       // connection

  // johnfitz -- reset developer stats
  memset(&dev_stats, 0, sizeof(dev_stats));
  memset(&dev_peakstats, 0, sizeof(dev_peakstats));
  memset(&dev_overflows, 0, sizeof(dev_overflows));
}

/*
==================
CL_ParseUpdate

Parse an entity update message from the server
If an entities model or origin changes from frame to frame, it must be
relinked.  Other attributes can change without relinking.
==================
*/
void CL_ParseUpdate(int bits) {
  int i;
  qmodel_t *model;
  int modnum;
  qboolean forcelink;
  entity_t *ent;
  int num;
  int skin;

  if (CLS_GetSignon() == SIGNONS - 1) {
    // first update is the final signon stage
    CLS_SetSignon(SIGNONS);
    CL_SignonReply();
  }

  if (bits & U_MOREBITS) {
    i = CL_MSG_ReadByte();
    bits |= (i << 8);
  }

  // johnfitz -- PROTOCOL_FITZQUAKE
  if (CL_Protocol() == PROTOCOL_FITZQUAKE || CL_Protocol() == PROTOCOL_RMQ) {
    if (bits & U_EXTEND1) {
      bits |= CL_MSG_ReadByte() << 16;
    }
    if (bits & U_EXTEND2) {
      bits |= CL_MSG_ReadByte() << 24;
    }
  }
  // johnfitz

  if (bits & U_LONGENTITY) {
    num = CL_MSG_ReadShort();
  } else {
    num = CL_MSG_ReadByte();
  }

  ent = CL_EntityNum(num);

  if (ent->msgtime != CL_MTimeOld())
    forcelink = true;  // no previous frame to lerp from
  else
    forcelink = false;

  // johnfitz -- lerping
  if (ent->msgtime + 0.2 < CL_MTime())  // more than 0.2 seconds since the last
                                        // message (most entities think every
                                        // 0.1 sec)
    ent->lerpflags |= LERP_RESETANIM;   // if we missed a think, we'd be lerping
                                        // from the wrong frame
  // johnfitz

  ent->msgtime = CL_MTime();

  if (bits & U_MODEL) {
    modnum = CL_MSG_ReadByte();
    if (modnum >= MAX_MODELS) Host_Error("CL_ParseModel: bad modnum");
  } else
    modnum = ent->baseline.modelindex;

  if (bits & U_FRAME)
    ent->frame = CL_MSG_ReadByte();
  else
    ent->frame = ent->baseline.frame;

  if (bits & U_COLORMAP)
    CL_MSG_ReadByte();  // colormap -- no idea what this is good for
  if (bits & U_SKIN)
    skin = CL_MSG_ReadByte();
  else
    skin = ent->baseline.skin;
  if (skin != ent->skinnum) {
    ent->skinnum = skin;
    if (num > 0 && num <= CL_MaxClients()) R_TranslateNewPlayerSkin(num - 1);
  }
  if (bits & U_EFFECTS)
    ent->effects = CL_MSG_ReadByte();
  else
    ent->effects = 0;

  // shift the known values for interpolation
  VectorCopy(ent->msg_origins[0], ent->msg_origins[1]);
  VectorCopy(ent->msg_angles[0], ent->msg_angles[1]);

  if (bits & U_ORIGIN1)
    ent->msg_origins[0][0] = CL_MSG_ReadCoord();
  else
    ent->msg_origins[0][0] = ent->baseline.origin[0];
  if (bits & U_ANGLE1)
    ent->msg_angles[0][0] = CL_MSG_ReadAngle(CL_ProtocolFlags());
  else
    ent->msg_angles[0][0] = ent->baseline.angles[0];

  if (bits & U_ORIGIN2)
    ent->msg_origins[0][1] = CL_MSG_ReadCoord();
  else
    ent->msg_origins[0][1] = ent->baseline.origin[1];
  if (bits & U_ANGLE2)
    ent->msg_angles[0][1] = CL_MSG_ReadAngle(CL_ProtocolFlags());
  else
    ent->msg_angles[0][1] = ent->baseline.angles[1];

  if (bits & U_ORIGIN3)
    ent->msg_origins[0][2] = CL_MSG_ReadCoord();
  else
    ent->msg_origins[0][2] = ent->baseline.origin[2];
  if (bits & U_ANGLE3)
    ent->msg_angles[0][2] = CL_MSG_ReadAngle(CL_ProtocolFlags());
  else
    ent->msg_angles[0][2] = ent->baseline.angles[2];

  // johnfitz -- lerping for movetype_step entities
  if (bits & U_STEP) {
    ent->lerpflags |= LERP_MOVESTEP;
    ent->forcelink = true;
  } else
    ent->lerpflags &= ~LERP_MOVESTEP;
  // johnfitz

  // johnfitz -- PROTOCOL_FITZQUAKE and PROTOCOL_NEHAHRA
  if (CL_Protocol() == PROTOCOL_FITZQUAKE || CL_Protocol() == PROTOCOL_RMQ) {
    if (bits & U_ALPHA)
      ent->alpha = CL_MSG_ReadByte();
    else
      ent->alpha = ent->baseline.alpha;
    if (bits & U_SCALE) CL_MSG_ReadByte();  // PROTOCOL_RMQ: currently ignored
    if (bits & U_FRAME2)
      ent->frame = (ent->frame & 0x00FF) | (CL_MSG_ReadByte() << 8);
    if (bits & U_MODEL2) modnum = (modnum & 0x00FF) | (CL_MSG_ReadByte() << 8);
    if (bits & U_LERPFINISH) {
      ent->lerpfinish = ent->msgtime + ((float)(CL_MSG_ReadByte()) / 255);
      ent->lerpflags |= LERP_FINISH;
    } else
      ent->lerpflags &= ~LERP_FINISH;
  } else if (CL_Protocol() == PROTOCOL_NETQUAKE) {
    // HACK: if this bit is set, assume this is PROTOCOL_NEHAHRA
    if (bits & U_TRANS) {
      float a, b;

      if (warn_about_nehahra_protocol) {
        Con_Warning("nonstandard update bit, assuming Nehahra protocol\n");
        warn_about_nehahra_protocol = false;
      }

      a = CL_MSG_ReadFloat();
      b = CL_MSG_ReadFloat();          // alpha
      if (a == 2) CL_MSG_ReadFloat();  // fullbright (not using this yet)
      ent->alpha = ENTALPHA_ENCODE(b);
    } else
      ent->alpha = ent->baseline.alpha;
  }
  // johnfitz

  // johnfitz -- moved here from above
  model = cl.model_precache[modnum];
  if (model != ent->model) {
    ent->model = model;
    // automatic animation (torches, etc) can be either all together
    // or randomized
    if (model) {
      if (model->synctype == ST_RAND)
        ent->syncbase = (float)(rand() & 0x7fff) / 0x7fff;
      else
        ent->syncbase = 0.0;
    } else
      forcelink = true;  // hack to make null model players work
    if (num > 0 && num <= CL_MaxClients()) R_TranslateNewPlayerSkin(num - 1);

    ent->lerpflags |= LERP_RESETANIM;  // johnfitz -- don't lerp animation
                                       // across model changes
  }
  // johnfitz

  if (forcelink) {  // didn't have an update last message
    VectorCopy(ent->msg_origins[0], ent->msg_origins[1]);
    VectorCopy(ent->msg_origins[0], ent->origin);
    VectorCopy(ent->msg_angles[0], ent->msg_angles[1]);
    VectorCopy(ent->msg_angles[0], ent->angles);
    ent->forcelink = true;
  }
}

/*
=====================
CL_NewTranslation
=====================
*/
void CL_NewTranslation(int slot) {
  if (slot > CL_MaxClients()) {
    Go_Error("CL_NewTranslation: slot > cl.maxClients");
  }
  R_TranslatePlayerSkin(slot);
}

/*
=====================
CL_ParseStatic
=====================
*/
void CL_ParseStatic(int version)  // johnfitz -- added a parameter
{
  // TODO(therjak): this can be moved to go if R_AddEfrags, CL_ParseBaseline,
  // and cl?
  entity_t *ent;
  int i;

  i = CL_num_statics();
  if (i >= MAX_STATIC_ENTITIES) Host_Error("Too many static entities");

  ent = &cl_static_entities[i];
  Inc_CL_num_statics();
  CL_ParseBaselineS(i, version);  // johnfitz -- added second parameter

  // copy it to the current state

  ent->model = cl.model_precache[ent->baseline.modelindex];
  ent->lerpflags |= LERP_RESETANIM;  // johnfitz -- lerping
  ent->frame = ent->baseline.frame;

  ent->skinnum = ent->baseline.skin;
  ent->effects = 0;
  ent->alpha = ent->baseline.alpha;  // johnfitz -- alpha

  VectorCopy(ent->baseline.origin, ent->origin);
  VectorCopy(ent->baseline.angles, ent->angles);
  R_AddEfrags(ent);
}


