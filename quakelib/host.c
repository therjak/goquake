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


int HostClient(void) { return Host_Client(); }

jmp_buf host_abortserver;

byte *host_colormap;

cvar_t host_speeds;
cvar_t host_maxfps;
cvar_t host_timescale;
cvar_t max_edicts;
cvar_t serverprofile;
cvar_t teamplay;
cvar_t samelevel;
cvar_t skill;
cvar_t deathmatch;
cvar_t coop;
cvar_t developer;
cvar_t temp1;
cvar_t devstats;

devstats_t dev_stats, dev_peakstats;
overflowtimes_t dev_overflows;  // this stores the last time overflow messages
                                // were displayed, not the last time overflows
                                // occured

/*
================
Host_EndGame
================
*/
void Host_EndGame(const char *message) {
  Con_DPrintf("Host_EndGame: %s\n", message);

  if (SV_Active()) Host_ShutdownServer(false);

  if (CLS_GetState() == ca_dedicated)
    Go_Error_S("Host_EndGame: %v\n", message);  // dedicated servers exit

  if (!CLS_IsDemoCycleStopped())
    CL_NextDemo();
  else
    CL_Disconnect();

  longjmp(host_abortserver, 1);
}

/*
================
Host_Error

This shuts down both the client and server
================
*/
void Host_Error(const char *error, ...) {
  va_list argptr;
  char string[1024];
  static qboolean inerror = false;

  if (inerror) Go_Error("Host_Error: recursively entered");
  inerror = true;

  SCR_EndLoadingPlaque();  // reenable screen updates

  va_start(argptr, error);
  q_vsnprintf(string, sizeof(string), error, argptr);
  va_end(argptr);
  Con_Printf("Host_Error: %s\n", string);

  if (SV_Active()) Host_ShutdownServer(false);

  if (CLS_GetState() == ca_dedicated)
    Go_Error_S("Host_Error: %v\n", string);  // dedicated servers exit

  CL_Disconnect();
  CLS_StopDemoCycle();
  CL_SetIntermission(0);  // johnfitz -- for errors during intermissions
                          // (changelevel with no map found, etc.)

  inerror = false;

  longjmp(host_abortserver, 1);
}

/*
=================
SV_BroadcastPrintf

Sends text to all active clients
=================
*/

void SV_BroadcastPrintf(const char *fmt, ...) {
  va_list argptr;
  char string[1024];

  va_start(argptr, fmt);
  q_vsnprintf(string, sizeof(string), fmt, argptr);
  va_end(argptr);
  SV_BroadcastPrint2(string);
}

/*
=======================
Host_InitLocal
======================
*/
void Host_InitLocal(void) {
  Host_InitCommands();

  Cvar_FakeRegister(&host_speeds, "host_speeds");
  Cvar_FakeRegister(&host_timescale, "host_timescale");
  Cvar_FakeRegister(&max_edicts, "max_edicts");
  Cvar_FakeRegister(&devstats, "devstats");
  Cvar_FakeRegister(&serverprofile, "serverprofile");
  Cvar_FakeRegister(&teamplay, "teamplay");
  Cvar_FakeRegister(&samelevel, "samelevel");
  Cvar_FakeRegister(&skill, "skill");
  Cvar_FakeRegister(&developer, "developer");
  Cvar_FakeRegister(&coop, "coop");
  Cvar_FakeRegister(&deathmatch, "deathmatch");
  Cvar_FakeRegister(&temp1, "temp1");

  Host_FindMaxClients();
}

/*
===============
Host_WriteConfiguration

Writes key bindings and archived cvars to config.cfg
===============
*/
void Host_WriteConfiguration(void) {
  FILE *f;

  // dedicated servers initialize the host but don't parse and set the
  // config.cfg cvars
  if (host_initialized & !CMLDedicated()) {
    f = fopen(va("%s/config.cfg", Com_Gamedir()), "w");
    if (!f) {
      Con_Printf("Couldn't write config.cfg.\n");
      return;
    }

    VID_SyncCvars();  // johnfitz -- write actual current mode to config file,
                      // in case cvars were messed with

    Key_WriteBindings(f);
    Cvar_WriteVariables(f);

    // johnfitz -- extra commands to preserve state
    fprintf(f, "vid_restart\n");
    if (CL_KeyMLookDown()) fprintf(f, "+mlook\n");
    // johnfitz

    fclose(f);
  }
}

/*
=================
SV_ClientPrintf

Sends text across to be displayed
FIXME: make this just a stuffed echo?
=================
*/
void SV_ClientPrintf2(int client, const char *fmt, ...) {
  va_list argptr;
  char string[1024];

  va_start(argptr, fmt);
  q_vsnprintf(string, sizeof(string), fmt, argptr);
  va_end(argptr);
  SV_ClientPrint2(client, string);
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
  /* host_hunklevel MUST be set at this point */
  Hunk_FreeToLowMark(host_hunklevel);
  CLS_SetSignon(0);
  FreeEdicts(sv.edicts);
  SV_Clear();
  CL_Clear();
  memset(&sv, 0, sizeof(sv));
  memset(&cl, 0, sizeof(cl));
}

//==============================================================================
//
// Host Frame
//
//==============================================================================

/*
===================
Host_GetConsoleCommands

Add them exactly as if they had been typed at the console
===================
*/
void Host_GetConsoleCommands(void) {
  const char *cmd;

  if (!CMLDedicated()) return;  // no stdin necessary in graphical mode

  while (1) {
    cmd = Sys_ConsoleInput();
    if (!cmd) break;
    Cbuf_AddText(cmd);
  }
}

/*
==================
Host_ServerFrame
==================
*/
void Host_ServerFrame(void) {
  int i, active;  // johnfitz

  // run the world state
  Set_pr_global_struct_frametime(HostFrameTime());

  // set the time and clear the general datagram
  SV_ClearDatagram();

  // check for new clients
  SV_CheckForNewClients();

  // read client messages
  SV_RunClients();

  // move things around and think
  // always pause in single player if in console or menus
  if (!SV_Paused() && (SVS_GetMaxClients() > 1 || GetKeyDest() == key_game))
    SV_Physics();

  // johnfitz -- devstats
  if (CLS_GetSignon() == SIGNONS) {
    for (i = 0, active = 0; i < SV_NumEdicts(); i++) {
      if (!EDICT_FREE(i)) active++;
    }
    if (active > 600 && dev_peakstats.edicts <= 600)
      Con_DWarning("%i edicts exceeds standard limit of 600.\n", active);
    dev_stats.edicts = active;
    dev_peakstats.edicts = q_max(active, dev_peakstats.edicts);
  }
  // johnfitz

  // send all messages to the clients
  SV_SendClientMessages();
}

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

  if (setjmp(host_abortserver))
    return;  // something bad happened, or the server disconnected

  // keep the random time dependent
  rand();

  // decide the simulation time
  if (!Host_FilterTime())
    return;  // don't run too fast, or packets will flood out

  // get new key events
  Key_UpdateForDest();
  IN_UpdateInputMode();
  IN_SendKeyEvents();

  // process console commands
  Cbuf_Execute();

  NET_Poll();

  // if running the server locally, make intentions now
  if (SV_Active()) CL_SendCmd();

  //-------------------
  //
  // server operations
  //
  //-------------------

  // check for commands typed to the host
  Host_GetConsoleCommands();

  if (SV_Active()) Host_ServerFrame();

  //-------------------
  //
  // client operations
  //
  //-------------------

  // if running the server remotely, send intentions now after
  // the incoming messages have been read
  if (!SV_Active()) CL_SendCmd();

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

void Host_Frame() {
  double time1, time2;
  static double timetotal;
  static int timecount;
  int i, c, m;

  if (!Cvar_GetValue(&serverprofile)) {
    _Host_Frame();
    return;
  }

  time1 = Sys_DoubleTime();
  _Host_Frame();
  time2 = Sys_DoubleTime();

  timetotal += time2 - time1;
  timecount++;

  if (timecount < 1000) return;

  m = timetotal * 1000 / timecount;
  timecount = 0;
  timetotal = 0;
  c = 0;
  for (i = 0; i < SVS_GetMaxClients(); i++) {
    if (GetClientActive(i)) c++;
  }

  Con_Printf("serverprofile: %2i clients %2i msec\n", c, m);
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
  if (CLS_GetState() != ca_dedicated) {
    Key_Init();
    Con_Init();
  }
  PR_Init();
  Mod_Init();
  NET_Init();
  SV_Init();

  Con_Printf("Exe: " __TIME__ " " __DATE__ "\n");
  Con_Printf("%4.1f megabyte heap\n", host_parms->memsize / (1024 * 1024.0));

  if (CLS_GetState() != ca_dedicated) {
    int length = 0;
    host_colormap = (byte *)COM_LoadFileGo("gfx/colormap.lmp", &length);
    if (!host_colormap) Go_Error("Couldn't load gfx/colormap.lmp");

    V_Init();
    Chase_Init();
    ExtraMaps_Init();  // johnfitz
    Modlist_Init();    // johnfitz
    DemoList_Init();   // ericw
    VID_Init();
    TexMgr_Init();  // johnfitz
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
    if (con_initialized) History_Shutdown();
    S_Shutdown();
    VID_Shutdown();
  }
}
