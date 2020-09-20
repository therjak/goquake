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

  ClearTempEntities();

  srand((int)(CL_Time() * 1000));  // johnfitz -- freeze beams when paused

  // update lightning
  for (i = 0, b = cl_beams; i < MAX_BEAMS; i++, b++) {
    if (!b->model || b->endtime < CL_Time()) continue;

    // if coming from the player, update the start position
    if (b->entity == CLViewentityNum()) {
      VectorCopy(CLViewEntity()->origin, b->start);
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

void CLPrecacheModel(const char* cn, int i) {
  cl.model_precache[i] = Mod_ForName(cn, false);
}

void FinishCL_ParseServerInfo(void) {
  // local state
  cl.worldmodel = cl.model_precache[1];
  SetWorldEntityModel(cl.worldmodel);
  
  R_NewMap();

  Hunk_Check();  // make sure nothing is hurt
  noclip_anglehack = false;  // noclip is turned off at start
  warn_about_nehahra_protocol = true;  // johnfitz -- warn about nehahra
                                       // protocol hack once per server
                                       // connection
}

void CL_ParseUpdate(int num, int modnum) {
  qmodel_t *model;
  qboolean forcelink;
  entity_t *ent;
  ent = CL_EntityNum(num);
  model = cl.model_precache[modnum];
    if (!model) {
      Con_Warning("no model %i\n", modnum);
    }
  if (model != ent->model) {
    ent->model = model;
  }
}

void CL_ParseStaticC(entity_t* ent, int modelindex)  {
  ent->model = cl.model_precache[modelindex];
}
