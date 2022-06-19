// SPDX-License-Identifier: GPL-2.0-or-later
// host.c -- coordinates spawning and killing of local servers

#include <setjmp.h>

#include "quakedef.h"

/*

A server can allways be started, even if the system started out as a client
to a remote system.

A client can NOT be started if the system started as a dedicated server.

Memory is cleared / released when a server or client begins, not when they end.

*/

quakeparms_t *host_parms;

int host_hunklevel;

cvar_t max_edicts;
cvar_t developer;

void VID_Init(void);

/*
================
Host_Error

This shuts down both the client and server
================
*/
void Host_Error(const char *error, ...) {
  va_list argptr;
  char string[1024];
  va_start(argptr, error);
  q_vsnprintf(string, sizeof(string), error, argptr);
  va_end(argptr);
  GoHostError(string);
}

/*
================
Host_ClearMemory

This clears all the memory used by both the client and server, but does
not reinitialize anything.
================
*/
void Host_ClearMemory(void) {
  Con_DPrintf("Clearing memory\n");
  Mod_ClearAll();
  ModClearAllGo();
  /* host_hunklevel MUST be set at this point */
  Hunk_FreeToLowMark(host_hunklevel);
  memset(&cl, 0, sizeof(cl));
}

/*
====================
Host_Init
====================
*/
void Host_Init(void) {
  int minimum_memory;
  if (CMLStandardQuake())
    minimum_memory = MINIMUM_MEMORY;
  else
    minimum_memory = MINIMUM_MEMORY_LEVELPAK;

  if (CMLMinMemory())
    host_parms->memsize = minimum_memory;

  if (host_parms->memsize < minimum_memory)
    Sys_Error("Only %4.1f megs of memory available, can't execute game",
              host_parms->memsize / (float)0x100000);

  Memory_Init(host_parms->membase, host_parms->memsize);
  Cvar_FakeRegister(&max_edicts, "max_edicts");
  Cvar_FakeRegister(&developer, "developer");
}

void HostInitAllocEnd() {
  Hunk_AllocName(0, "-HOST_HUNKLEVEL-");
  host_hunklevel = Hunk_LowMark();
}
