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

qboolean noclip_anglehack;

/*
================
Host_ClearMemory

This clears all the memory used by both the client and server, but does
not reinitialize anything.
================
*/
void Host_ClearMemory(void) {
  Con_DPrintf("Clearing memory\n");
  D_FlushCaches();
  Mod_ClearAll();
  ModClearAllGo();
  /* host_hunklevel MUST be set at this point */
  Hunk_FreeToLowMark(host_hunklevel);
  CLS_SetSignon(0);
  FreeEdicts();
  SV_Clear();
  CL_Clear();
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

  if (CMLMinMemory()) host_parms->memsize = minimum_memory;

  if (host_parms->memsize < minimum_memory)
    Sys_Error("Only %4.1f megs of memory available, can't execute game",
              host_parms->memsize / (float)0x100000);

  Memory_Init(host_parms->membase, host_parms->memsize);
  COM_InitFilesystem();
  Cvar_FakeRegister(&max_edicts, "max_edicts");
  Cvar_FakeRegister(&developer, "developer");
  Host_FindMaxClients();
  Go_LoadWad();
  if (CLS_GetState() != ca_dedicated) {
    Key_Init();
    Con_Init();
  }
  Mod_Init();
  NET_Init();
  SV_Init();

  Con_Printf("Exe: " __TIME__ " " __DATE__ "\n");
  Con_Printf("%4.1f megabyte heap\n", host_parms->memsize / (1024 * 1024.0));

  if (CLS_GetState() != ca_dedicated) {
    int length = 0;

    // ExtraMaps_Init();  // johnfitz
    // Modlist_Init();    // johnfitz
    // DemoList_Init();   // ericw
    VID_Init();
    TexMgrInit();  // johnfitz
    Draw_Init();
    SCR_Init();
    R_Init();
    S_Init();
    Sbar_Init();
    CL_Init();
  }

  Hunk_AllocName(0, "-HOST_HUNKLEVEL-");
  host_hunklevel = Hunk_LowMark();

  HostSetInitialized();
  Con_Printf("\n========= Quake Initialized =========\n\n");

  if (CLS_GetState() != ca_dedicated) {
    Cbuf_InsertText("exec quake.rc\n");
    // johnfitz -- in case the vid mode was locked during vid_init, we can
    // unlock it now.
    // note: two leading newlines because the command buffer swallows one of
    // them.
    Cbuf_AddText("\n\nvid_unlock\n");
  }

  if (CLS_GetState() == ca_dedicated) {
    Cbuf_AddText("exec autoexec.cfg\n");
    Cbuf_AddText("stuffcmds");
    Cbuf_Execute();
    if (!SV_Active()) Cbuf_AddText("map start\n");
  }
}
