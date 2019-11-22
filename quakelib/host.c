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

qboolean host_initialized;  // true if into command execution

int host_framecount;

int host_hunklevel;

int minimum_memory;

jmp_buf host_abortserver;

cvar_t host_speeds;
cvar_t host_maxfps;
cvar_t host_timescale;
cvar_t max_edicts;
cvar_t serverprofile;
cvar_t teamplay;
cvar_t samelevel;
cvar_t skill;
cvar_t developer;
cvar_t devstats;
cvar_t sv_gravity;

devstats_t dev_stats, dev_peakstats;
overflowtimes_t dev_overflows;  // this stores the last time overflow messages
                                // were displayed, not the last time overflows
                                // occured

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
=======================
Host_InitLocal
======================
*/
void Host_InitLocal(void) {
  // Host_InitCommands();

  Cvar_FakeRegister(&host_speeds, "host_speeds");
  Cvar_FakeRegister(&host_timescale, "host_timescale");
  Cvar_FakeRegister(&max_edicts, "max_edicts");
  Cvar_FakeRegister(&devstats, "devstats");
  Cvar_FakeRegister(&serverprofile, "serverprofile");
  Cvar_FakeRegister(&teamplay, "teamplay");
  Cvar_FakeRegister(&samelevel, "samelevel");
  Cvar_FakeRegister(&skill, "skill");
  Cvar_FakeRegister(&developer, "developer");
  Cvar_FakeRegister(&sv_gravity, "sv_gravity");

  Host_FindMaxClients();
}

/*
===============
Host_WriteConfiguration

Writes key bindings and archived cvars to config.cfg
===============
*/
void Host_WriteConfiguration(void) {
  if (!host_initialized) {
    return;
  }
  // everything else is done in go
  HostWriteConfiguration();
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

//==============================================================================
//
// Host Frame
//
//==============================================================================

/*
==================
Host_Frame

Runs all active servers
==================
*/
void _Host_Frame() {
  static double time1 = 0;
  static double time2 = 0;
  static double time3 = 0;
  int pass1, pass2, pass3;

  // keep the random time dependent
  rand();  // to keep the c side happy

  // fetch results from server
  if (CLS_GetState() == ca_connected) CL_ReadFromServer();

  // update video
  if (Cvar_GetValue(&host_speeds)) time1 = Sys_DoubleTime();

  SCR_UpdateScreen();

  CL_RunParticles();  // johnfitz -- seperated from rendering

  if (Cvar_GetValue(&host_speeds)) time2 = Sys_DoubleTime();

  // update audio
  // adds music raw samples and/or advances midi driver
  if (CLS_GetSignon() == SIGNONS) {
    S_Update(r_origin, vpn, vright, vup);
    CL_DecayLights();
  } else
    S_Update(vec3_origin, vec3_origin, vec3_origin, vec3_origin);

  if (Cvar_GetValue(&host_speeds)) {
    pass1 = (time1 - time3) * 1000;
    time3 = Sys_DoubleTime();
    pass2 = (time2 - time1) * 1000;
    pass3 = (time3 - time2) * 1000;
    Con_Printf("%3i tot %3i server %3i gfx %3i snd\n", pass1 + pass2 + pass3,
               pass1, pass2, pass3);
  }

  host_framecount++;
}

/*
====================
Host_Init
====================
*/
void Host_Init(void) {
  if (CMLStandardQuake())
    minimum_memory = MINIMUM_MEMORY;
  else
    minimum_memory = MINIMUM_MEMORY_LEVELPAK;

  if (CMLMinMemory()) host_parms->memsize = minimum_memory;

  if (host_parms->memsize < minimum_memory)
    Sys_Error("Only %4.1f megs of memory available, can't execute game",
              host_parms->memsize / (float)0x100000);

  Memory_Init(host_parms->membase, host_parms->memsize);
  Cvar_Init();  // johnfitz
  COM_InitFilesystem();
  Host_InitLocal();
  W_LoadWadFile();  // johnfitz -- filename is now hard-coded for honesty
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

    V_Init();
    Chase_Init();
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

  host_initialized = true;
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

/*
===============
Host_Shutdown

FIXME: this is a callback from Sys_Quit and Sys_Error.  It would be better
to run quit through here before the final handoff to the sys code.
===============
*/
void Host_Shutdown(void) {
  static qboolean isdown = false;

  if (isdown) {
    printf("recursive shutdown\n");
    return;
  }
  isdown = true;

  // keep Con_Printf from trying to update the screen
  SetScreenDisabled(true);

  Host_WriteConfiguration();

  NET_Shutdown();

  if (CLS_GetState() != ca_dedicated) {
    if (Con_Initialized()) History_Shutdown();
    S_Shutdown();
    VID_Shutdown();
  }
}
