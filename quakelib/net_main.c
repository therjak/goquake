/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2010-2014 QuakeSpasm developers

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

*/

#include "_cgo_export.h"
//
#include "q_stdinc.h"
//
#include "arch_def.h"
//
#include "net_sys.h"
//
#include "quakedef.h"
//
#include "net_defs.h"

int net_activeSockets = -1;
int net_freeSockets = -1;
int net_numsockets = 0;

int net_hostport;
int DEFAULTnet_hostport = 26000;

char my_tcpip_address[NET_NAMELEN];

static qboolean listening = false;

qboolean slistInProgress = false;
qboolean slistSilent = false;
qboolean slistLocal = true;
static double slistStartTime;
static int slistLastShown;

static void Slist_Send(void *);
static void Slist_Poll(void *);
static PollProcedure slistSendProcedure = {NULL, 0.0, Slist_Send};
static PollProcedure slistPollProcedure = {NULL, 0.0, Slist_Poll};

int messagesSent = 0;
int messagesReceived = 0;
int unreliableMessagesSent = 0;
int unreliableMessagesReceived = 0;

static cvar_t net_messagetimeout;
cvar_t hostname;

// these two macros are to make the code more readable
#define sfunc net_drivers[sock->driver]
#define dfunc net_drivers[net_driverlevel]

int net_driverlevel;

// convert
int NET_NewQSocket(void) { return -1; }

// convert
void NET_FreeQSocket(int sock) {}

const char *NET_QSocketGetAddressString(int s) {
  static char buffer[NET_NAMELEN];
  ClientAddress(s, &buffer[0], NET_NAMELEN);
  return &buffer[0];
}

// convert
static void PrintSlistHeader(void) {
  Con_Printf("Server          Map             Users\n");
  Con_Printf("--------------- --------------- -----\n");
  slistLastShown = 0;
}

void NET_Slist_f(void) {
  if (slistInProgress) return;

  if (!slistSilent) {
    Con_Printf("Looking for Quake servers...\n");
    PrintSlistHeader();
  }

  slistInProgress = true;
  slistStartTime = Sys_DoubleTime();

  SchedulePollProcedure(&slistSendProcedure, 0.0);
  SchedulePollProcedure(&slistPollProcedure, 0.1);

  hostCacheCount = 0;
}

void NET_SlistSort(void) {
  if (hostCacheCount > 1) {
    int i, j;
    hostcache_t temp;
    for (i = 0; i < hostCacheCount; i++) {
      for (j = i + 1; j < hostCacheCount; j++) {
        if (strcmp(hostcache[j].name, hostcache[i].name) < 0) {
          memcpy(&temp, &hostcache[j], sizeof(hostcache_t));
          memcpy(&hostcache[j], &hostcache[i], sizeof(hostcache_t));
          memcpy(&hostcache[i], &temp, sizeof(hostcache_t));
        }
      }
    }
  }
}

const char *NET_SlistPrintServer(int idx) {
  static char string[64];

  if (idx < 0 || idx >= hostCacheCount) return "";

  if (hostcache[idx].maxusers) {
    q_snprintf(string, sizeof(string), "%-15.15s %-15.15s %2u/%2u\n",
               hostcache[idx].name, hostcache[idx].map, hostcache[idx].users,
               hostcache[idx].maxusers);
  } else {
    q_snprintf(string, sizeof(string), "%-15.15s %-15.15s\n",
               hostcache[idx].name, hostcache[idx].map);
  }

  return string;
}

const char *NET_SlistPrintServerName(int idx) {
  if (idx < 0 || idx >= hostCacheCount) return "";
  return hostcache[idx].cname;
}

static void Slist_Send(void *unused) { REPORT_BadCall(); }

static void Slist_Poll(void *unused) { REPORT_BadCall(); }

int hostCacheCount = 0;
hostcache_t hostcache[HOSTCACHESIZE];

void NET_Init(void) {
  DEFAULTnet_hostport = CMLPort();
  net_hostport = DEFAULTnet_hostport;

  net_numsockets = SVS_GetMaxClientsLimit();
  if (CLS_GetState() != ca_dedicated) net_numsockets++;
  if (CMLListen() || CLS_GetState() == ca_dedicated) listening = true;

  NET_SetTime();

  Cvar_FakeRegister(&net_messagetimeout, "net_messagetimeout");
  Cvar_FakeRegister(&hostname, "hostname");

  Cmd_AddCommand("slist", NET_Slist_f);

  if (*my_tcpip_address) {
    Con_DPrintf("TCP/IP address %s\n", my_tcpip_address);
  }
}

static PollProcedure *pollProcedureList = NULL;

void NET_Poll(void) {
  PollProcedure *pp;

  NET_SetTime();

  for (pp = pollProcedureList; pp; pp = pp->next) {
    if (pp->nextTime > NET_GetTime()) break;
    pollProcedureList = pp->next;
    pp->procedure(pp->arg);
  }
}

void SchedulePollProcedure(PollProcedure *proc, double timeOffset) {
  PollProcedure *pp, *prev;

  proc->nextTime = Sys_DoubleTime() + timeOffset;
  for (pp = pollProcedureList, prev = NULL; pp; pp = pp->next) {
    if (pp->nextTime >= proc->nextTime) break;
    prev = pp;
  }

  if (prev == NULL) {
    proc->next = pollProcedureList;
    pollProcedureList = proc;
    return;
  }

  proc->next = pp;
  prev->next = proc;
}

//
// reading functions
//

const char *MSG_ReadString(void) {
  static char string[2048];
  int c;
  size_t l;

  l = 0;
  do {
    c = MSG_ReadByte();
    if (c == -1 || c == 0) break;
    string[l] = c;
    l++;
  } while (l < sizeof(string) - 1);

  string[l] = 0;
  return string;
}

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
