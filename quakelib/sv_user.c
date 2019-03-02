// sv_user.c -- server code for moving users

#include "quakedef.h"

extern cvar_t sv_friction;
extern cvar_t sv_stopspeed;

static vec3_t forward, right, up;

// world
float *angles;
float *origin;
float *velocity;

qboolean onground;

cvar_t sv_idealpitchscale;
cvar_t sv_edgefriction;
cvar_t sv_altnoclip;

edict_t *SV_GetEdict(int cl) { return EDICT_NUM(GetClientEdictId(cl)); }

void SV_SetEdictNum(int cl, int num) { SetClientEdictId(cl, num); }

/*
===============
SV_SetIdealPitch
===============
*/
#define MAX_FORWARD 6
void SV_SetIdealPitch(void) {
  float angleval, sinval, cosval;
  trace_t tr;
  vec3_t top, bottom;
  float z[MAX_FORWARD];
  int i, j;
  int step, dir, steps;

  if (!((int)EVars(SV_Player())->flags & FL_ONGROUND)) return;

  angleval = EVars(SV_Player())->angles[YAW] * M_PI * 2 / 360;
  sinval = sin(angleval);
  cosval = cos(angleval);

  for (i = 0; i < MAX_FORWARD; i++) {
    top[0] = EVars(SV_Player())->origin[0] + cosval * (i + 3) * 12;
    top[1] = EVars(SV_Player())->origin[1] + sinval * (i + 3) * 12;
    top[2] = EVars(SV_Player())->origin[2] + EVars(SV_Player())->view_ofs[2];

    bottom[0] = top[0];
    bottom[1] = top[1];
    bottom[2] = top[2] - 160;

    tr = SV_Move(top, vec3_origin, vec3_origin, bottom, 1, SV_Player());
    if (tr.allsolid) return;  // looking at a wall, leave ideal the way is was

    if (tr.fraction == 1) return;  // near a dropoff

    z[i] = top[2] + tr.fraction * (bottom[2] - top[2]);
  }

  dir = 0;
  steps = 0;
  for (j = 1; j < i; j++) {
    step = z[j] - z[j - 1];
    if (step > -ON_EPSILON && step < ON_EPSILON) continue;

    if (dir && (step - dir > ON_EPSILON || step - dir < -ON_EPSILON))
      return;  // mixed changes

    steps++;
    dir = step;
  }

  if (!dir) {
    EVars(SV_Player())->idealpitch = 0;
    return;
  }

  if (steps < 2) return;
  EVars(SV_Player())->idealpitch = -dir * Cvar_GetValue(&sv_idealpitchscale);
}

/*
==================
SV_UserFriction

==================
*/
void SV_UserFriction(edict_t *player) {
  float *vel;
  float speed, newspeed, control;
  vec3_t start, stop;
  float friction;
  trace_t trace;

  vel = velocity;

  speed = sqrt(vel[0] * vel[0] + vel[1] * vel[1]);
  if (!speed) return;

  // if the leading edge is over a dropoff, increase friction
  start[0] = stop[0] = origin[0] + vel[0] / speed * 16;
  start[1] = stop[1] = origin[1] + vel[1] / speed * 16;
  start[2] = origin[2] + EdictV(player)->mins[2];
  stop[2] = start[2] - 34;

  trace = SV_Move(start, vec3_origin, vec3_origin, stop, true,
                  NUM_FOR_EDICT(player));

  if (trace.fraction == 1.0)
    friction = Cvar_GetValue(&sv_friction) * Cvar_GetValue(&sv_edgefriction);
  else
    friction = Cvar_GetValue(&sv_friction);

  // apply friction
  control = speed < Cvar_GetValue(&sv_stopspeed) ? Cvar_GetValue(&sv_stopspeed)
                                                 : speed;
  newspeed = speed - HostFrameTime() * control * friction;

  if (newspeed < 0) newspeed = 0;
  newspeed /= speed;

  vel[0] = vel[0] * newspeed;
  vel[1] = vel[1] * newspeed;
  vel[2] = vel[2] * newspeed;
}

/*
==============
SV_Accelerate
==============
*/
cvar_t sv_maxspeed;
cvar_t sv_accelerate;
void SV_Accelerate(float wishspeed, const vec3_t wishdir) {
  int i;
  float addspeed, accelspeed, currentspeed;

  currentspeed = DotProduct(velocity, wishdir);
  addspeed = wishspeed - currentspeed;
  if (addspeed <= 0) return;
  accelspeed = Cvar_GetValue(&sv_accelerate) * HostFrameTime() * wishspeed;
  if (accelspeed > addspeed) accelspeed = addspeed;

  for (i = 0; i < 3; i++) velocity[i] += accelspeed * wishdir[i];
}

void SV_AirAccelerate(float wishspeed, vec3_t wishveloc) {
  int i;
  float addspeed, wishspd, accelspeed, currentspeed;

  wishspd = VectorNormalize(wishveloc);
  if (wishspd > 30) wishspd = 30;
  currentspeed = DotProduct(velocity, wishveloc);
  addspeed = wishspd - currentspeed;
  if (addspeed <= 0) return;
  accelspeed = Cvar_GetValue(&sv_accelerate) * wishspeed * HostFrameTime();
  if (accelspeed > addspeed) accelspeed = addspeed;

  for (i = 0; i < 3; i++) velocity[i] += accelspeed * wishveloc[i];
}

void DropPunchAngle(edict_t *player) {
  float len;

  len = VectorNormalize(EdictV(player)->punchangle);

  len -= 10 * HostFrameTime();
  if (len < 0) len = 0;
  VectorScale(EdictV(player)->punchangle, len, EdictV(player)->punchangle);
}

/*
===================
SV_WaterMove

===================
*/
void SV_WaterMove(edict_t *player, movecmd_t *cmd) {
  int i;
  vec3_t wishvel;
  float speed, newspeed, wishspeed, addspeed, accelspeed;

  //
  // user intentions
  //
  AngleVectors(EdictV(player)->v_angle, forward, right, up);

  for (i = 0; i < 3; i++)
    wishvel[i] = forward[i] * cmd->forwardmove + right[i] * cmd->sidemove;

  if (!cmd->forwardmove && !cmd->sidemove && !cmd->upmove)
    wishvel[2] -= 60;  // drift towards bottom
  else
    wishvel[2] += cmd->upmove;

  wishspeed = VectorLength(wishvel);
  if (wishspeed > Cvar_GetValue(&sv_maxspeed)) {
    VectorScale(wishvel, Cvar_GetValue(&sv_maxspeed) / wishspeed, wishvel);
    wishspeed = Cvar_GetValue(&sv_maxspeed);
  }
  wishspeed *= 0.7;

  //
  // water friction
  //
  speed = VectorLength(velocity);
  if (speed) {
    newspeed = speed - HostFrameTime() * speed * Cvar_GetValue(&sv_friction);
    if (newspeed < 0) newspeed = 0;
    VectorScale(velocity, newspeed / speed, velocity);
  } else
    newspeed = 0;

  //
  // water acceleration
  //
  if (!wishspeed) return;

  addspeed = wishspeed - newspeed;
  if (addspeed <= 0) return;

  VectorNormalize(wishvel);
  accelspeed = Cvar_GetValue(&sv_accelerate) * wishspeed * HostFrameTime();
  if (accelspeed > addspeed) accelspeed = addspeed;

  for (i = 0; i < 3; i++) velocity[i] += accelspeed * wishvel[i];
}

void SV_WaterJump(edict_t *player) {
  if (SV_Time() > EdictV(player)->teleport_time ||
      !EdictV(player)->waterlevel) {
    EdictV(player)->flags = (int)EdictV(player)->flags & ~FL_WATERJUMP;
    EdictV(player)->teleport_time = 0;
  }
  EdictV(player)->velocity[0] = EdictV(player)->movedir[0];
  EdictV(player)->velocity[1] = EdictV(player)->movedir[1];
}

/*
===================
SV_NoclipMove -- johnfitz

new, alternate noclip. old noclip is still handled in SV_AirMove
===================
*/
void SV_NoclipMove(edict_t *player, movecmd_t *cmd) {
  AngleVectors(EdictV(player)->v_angle, forward, right, up);

  velocity[0] = forward[0] * cmd->forwardmove + right[0] * cmd->sidemove;
  velocity[1] = forward[1] * cmd->forwardmove + right[1] * cmd->sidemove;
  velocity[2] = forward[2] * cmd->forwardmove + right[2] * cmd->sidemove;
  velocity[2] += cmd->upmove * 2;  // doubled to match running speed

  if (VectorLength(velocity) > Cvar_GetValue(&sv_maxspeed)) {
    VectorNormalize(velocity);
    VectorScale(velocity, Cvar_GetValue(&sv_maxspeed), velocity);
  }
}

/*
===================
SV_AirMove
===================
*/
void SV_AirMove(edict_t *player, movecmd_t *cmd) {
  int i;
  vec3_t wishvel, wishdir;
  float wishspeed;
  float fmove, smove;

  AngleVectors(EdictV(player)->angles, forward, right, up);

  fmove = cmd->forwardmove;
  smove = cmd->sidemove;

  // hack to not let you back into teleporter
  if (SV_Time() < EdictV(player)->teleport_time && fmove < 0) fmove = 0;

  for (i = 0; i < 3; i++) wishvel[i] = forward[i] * fmove + right[i] * smove;

  if ((int)EdictV(player)->movetype != MOVETYPE_WALK)
    wishvel[2] = cmd->upmove;
  else
    wishvel[2] = 0;

  VectorCopy(wishvel, wishdir);
  wishspeed = VectorNormalize(wishdir);
  if (wishspeed > Cvar_GetValue(&sv_maxspeed)) {
    VectorScale(wishvel, Cvar_GetValue(&sv_maxspeed) / wishspeed, wishvel);
    wishspeed = Cvar_GetValue(&sv_maxspeed);
  }

  if (EdictV(player)->movetype == MOVETYPE_NOCLIP) {  // noclip
    VectorCopy(wishvel, velocity);
  } else if (onground) {
    SV_UserFriction(player);
    SV_Accelerate(wishspeed, wishdir);
  } else {  // not on ground, so little effect on velocity
    SV_AirAccelerate(wishspeed, wishvel);
  }
}

/*
===================
SV_ClientThink

the move fields specify an intended velocity in pix/sec
the angle fields specify an exact angular motion in degrees
===================
*/
void SV_ClientThink(int client) {
  movecmd_t cmd;
  entvars_t *entv = EVars(GetClientEdictId(client));
  edict_t *player = SV_GetEdict(client);
  vec3_t v_angle;

  if (entv->movetype == MOVETYPE_NONE) return;

  onground = (int)entv->flags & FL_ONGROUND;

  origin = entv->origin;
  velocity = entv->velocity;

  DropPunchAngle(player);

  //
  // if dead, behave differently
  //
  if (entv->health <= 0) return;

  //
  // angles
  // show 1/3 the pitch angle and all the roll angle
  cmd = GetClientMoveCmd(client);
  angles = entv->angles;

  VectorAdd(entv->v_angle, entv->punchangle, v_angle);
  angles[ROLL] = V_CalcRoll(entv->angles, entv->velocity) * 4;
  if (!entv->fixangle) {
    angles[PITCH] = -v_angle[PITCH] / 3;
    angles[YAW] = v_angle[YAW];
  }

  if ((int)entv->flags & FL_WATERJUMP) {
    SV_WaterJump(player);
    return;
  }
  //
  // walk
  //
  // johnfitz -- alternate noclip
  if (entv->movetype == MOVETYPE_NOCLIP &&
      Cvar_GetValue(&sv_altnoclip)) {
    SV_NoclipMove(player, &cmd);
  } else if (entv->waterlevel >= 2 &&
             entv->movetype != MOVETYPE_NOCLIP) {
    SV_WaterMove(player, &cmd);
  } else {
    SV_AirMove(player, &cmd);
  }
  // johnfitz
}

/*
===================
SV_ReadClientMove
===================
*/
void SV_ReadClientMove(int client) {
  movecmd_t move;
  entvars_t *entv = EVars(GetClientEdictId(client));

  int i;
  vec3_t angle;
  int bits;

  // read ping time
  SetClientPingTime(client, GetClientNumPings(client) % NUM_PING_TIMES,
                    SV_Time() - MSG_ReadFloat());
  SetClientNumPings(client, (GetClientNumPings(client) + 1) % NUM_PING_TIMES);

  // read current angles
  for (i = 0; i < 3; i++)
    // johnfitz -- 16-bit angles for PROTOCOL_FITZQUAKE
    if (SV_Protocol() == PROTOCOL_NETQUAKE)
      angle[i] = MSG_ReadAngle();
    else
      angle[i] = MSG_ReadAngle16();
  // johnfitz

  VectorCopy(angle, entv->v_angle);

  // read movement
  move.forwardmove = MSG_ReadShort();
  move.sidemove = MSG_ReadShort();
  move.upmove = MSG_ReadShort();
  SetClientMoveCmd(client, move);

  // read buttons
  bits = MSG_ReadByte();
  entv->button0 = bits & 1;
  entv->button2 = (bits & 2) >> 1;

  i = MSG_ReadByte();
  if (i) entv->impulse = i;
}

/*
===================
SV_ReadClientMessage

Returns false if the client should be killed
===================
*/
qboolean SV_ReadClientMessage(int client) {
  int ret;
  int ccmd;
  const char *s;

  do {
  nextmsg:
    ret = ClientGetMessage(client);
    if (ret == -1) {
      Sys_Print("SV_ReadClientMessage: ClientGetMessage failed\n");
      return false;
    }
    if (!ret) return true;

    while (1) {
      if (!GetClientActive(client)) return false;  // a command caused an error

      if (MSG_BadRead()) {
        Sys_Print("SV_ReadClientMessage: badread\n");
        return false;
      }

      ccmd = MSG_ReadChar();

      switch (ccmd) {
        case -1:
          goto nextmsg;  // end of message

        default:
          Sys_Print("SV_ReadClientMessage: unknown command char\n");
          return false;

        case clc_nop:
          //				Sys_Print ("clc_nop\n");
          break;

        case clc_stringcmd:
          s = MSG_ReadString();
          ret = 0;
          if (q_strncasecmp(s, "status", 6) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "god", 3) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "notarget", 8) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "fly", 3) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "name", 4) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "noclip", 6) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "setpos", 6) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "say", 3) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "say_team", 8) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "tell", 4) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "color", 5) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "kill", 4) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "pause", 5) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "spawn", 5) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "begin", 5) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "prespawn", 8) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "kick", 4) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "ping", 4) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "give", 4) == 0)
            ret = 1;
          else if (q_strncasecmp(s, "ban", 3) == 0)
            ret = 1;

          if (ret == 1)
            Cmd_ExecuteString(s, src_client);
          else {
            char *name = GetClientName(client);
            Con_DPrintf("%s tried to %s\n", name, s);
            free(name);
          }
          break;

        case clc_disconnect:
          //				Sys_Print ("SV_ReadClientMessage:
          // client
          // disconnected\n");
          return false;

        case clc_move:
          SV_ReadClientMove(client);
          break;
      }
    }
  } while (ret == 1);

  return true;
}

/*
==================
SV_RunClients
==================
*/
void SV_RunClients(void) {
  int i;
  movecmd_t move = {0, 0, 0};

  for (i = 0, host_client = 0; i < SVS_GetMaxClients(); i++, host_client++) {
    if (!GetClientActive(HostClient())) continue;

    Set_SV_Player(GetClientEdictId(HostClient()));

    if (!SV_ReadClientMessage(HostClient())) {
      SV_DropClient(HostClient(), false);  // client misbehaved...
      continue;
    }

    if (!GetClientSpawned(HostClient())) {
      // clear client movement until a new packet is received
      SetClientMoveCmd(HostClient(), move);
      continue;
    }

    // always pause in single player if in console or menus
    if (!SV_Paused() && (SVS_GetMaxClients() > 1 || GetKeyDest() == key_game))
      SV_ClientThink(HostClient());
  }
}
