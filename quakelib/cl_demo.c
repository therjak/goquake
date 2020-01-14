#include "quakedef.h"

#define NET_MAXMESSAGE 64000

// THERJAK: this should be possible to move now

static void CL_FinishTimeDemo(void);

/*
==============================================================================

DEMO CODE

When a demo is playing back, all NET_SendMessages are skipped, and
NET_GetMessages are read from the demo file.

Whenever cl.time gets past the last received message, another message is
read from the demo file.
==============================================================================
*/

// from ProQuake: space to fill out the demo header for record at any time
// thea: demo recording disabled
// static byte demo_head[3][MAX_MSGLEN];
// static int demo_head_size[2];

/*
==============
CL_StopPlayback

Called when a demo file runs out, or the user starts a game
==============
*/
// public
void CL_StopPlayback(void) {
  if (!CLS_IsDemoPlayback()) return;

  fclose(cls.demofile);
  CLS_SetDemoPlayback(false);
  CLS_SetDemoPaused(false);
  cls.demofile = NULL;
  CLS_SetState(ca_disconnected);

  if (CLS_IsTimeDemo()) CL_FinishTimeDemo();
}

/*
====================
CL_WriteDemoMessage

Dumps the current net message, prefixed by the length and view angles
====================
*/
// thea: demo recording disabled
void CL_WriteDemoMessage(void) {
  /*
  int len;
  int i;
  float f;

  len = LittleLong(SB_GetCurSize(&net_message));
  fwrite(&len, 4, 1, cls.demofile);

  f = LittleFloat(CLPitch());
  fwrite(&f, 4, 1, cls.demofile);
  f = LittleFloat(CLYaw());
  fwrite(&f, 4, 1, cls.demofile);
  f = LittleFloat(CLRoll());
  fwrite(&f, 4, 1, cls.demofile);

  // write net_message to demofile
  fwrite(net_message.daTa, SB_GetCurSize(&net_message), 1, cls.demofile);
  fflush(cls.demofile);
  */
}

int CL_GetDemoMessage(void) {
  int r, i;
  float f;
  byte readbuf[NET_MAXMESSAGE];

  if (CLS_IsDemoPaused()) return 0;

  // decide if it is time to grab the next message
  if (CLS_GetSignon() == SIGNONS)  // always grab until fully connected
  {
    if (CLS_IsTimeDemo()) {
      if (host_framecount == CLS_GetTimeDemoLastFrame())
        return 0;  // already read this frame's message
      CLS_SetTimeDemoLastFrame(host_framecount);
      // if this is the second frame, grab the real td_starttime
      // so the bogus time on the first frame doesn't count
      if (host_framecount == CLS_GetTimeDemoStartFrame() + 1) {
        CLS_SetTimeDemoStartTime(HostRealTime());
      }
    } else if (/* cl.time > 0 && */ CL_Time() <= CL_MTime()) {
      return 0;  // don't need another message yet
    }
  }

  // get the next message
  int cursize = 0;
  fread(&cursize, 4, 1, cls.demofile);
  for (i = 0; i < 3; i++) {
    r = fread(&f, 4, 1, cls.demofile);
    SetCL_MViewAngles(1, i, CL_MViewAngles(0, i));
    SetCL_MViewAngles(0, i, LittleFloat(f));
  }

  cursize = LittleLong(cursize);
  if (cursize > NET_MAXMESSAGE) {
    Go_Error("Demo message > NET_MAXMESSAGE");
  }
  // read demofile to net_message
  r = fread(&readbuf, cursize, 1, cls.demofile);
  CL_MSG_Replace(&readbuf, cursize);
  if (r != 1) {
    CL_StopPlayback();
    return 0;
  }

  return 1;
}

/*
====================
CL_Stop_f

stop recording a demo
====================
*/
// public
void CL_Stop_f(void) {
  if (!IsSrcCommand()) return;

  // thea: demo recording is disabled
  //  if (!CLS_IsDemoRecording()) {
  Con_Printf("Not recording a demo.\n");
  return;
  /*  }

    // write a disconnect message to the demo file
    SZ_Clear(&net_message);
    MSG_WriteByte(&net_message, svc_disconnect);
    CL_WriteDemoMessage();

    // finish up
    fclose(cls.demofile);
    cls.demofile = NULL;
    CLS_SetDemoRecording(false);
    Con_Printf("Completed demo\n");

    // ericw -- update demo tab-completion list
    DemoList_Rebuild();
  */
}

/*
====================
CL_Record_f

record <demoname> <map> [cd track]
====================
*/
// public
void CL_Record_f(void) {
  //  int c;
  //  char name[MAX_OSPATH];
  //  int track;

  if (!IsSrcCommand()) return;

  // thea: disable demo recording
  Con_Printf("Can't record demo: disabled\n");
  return;

  /*
    if (CLS_IsDemoPlayback()) {
      Con_Printf("Can't record during demo playback\n");
      return;
    }

    if (CLS_IsDemoRecording()) CL_Stop_f();

    c = Cmd_Argc();
    if (c != 2 && c != 3 && c != 4) {
      Con_Printf("record <demoname> [<map> [cd track]]\n");
      return;
    }

    if (strstr(Cmd_Argv(1), "..")) {
      Con_Printf("Relative pathnames are not allowed.\n");
      return;
    }

    if (c == 2 && CLS_GetState() == ca_connected) {
  #if 0
                  Con_Printf("Can not record - already connected to
  server\nClient demo recording must be started before connecting\n"); return;
  #endif
      if (CLS_GetSignon() < 2) {
        Con_Printf("Can't record - try again when connected\n");
        return;
      }
    }

    // write the forced cd track number, or -1
    if (c == 4) {
      track = Cmd_ArgvAsInt(3);
      Con_Printf("Forcing CD track to %i\n", CLS_GetForceTrack());
    } else {
      track = -1;
    }

    q_snprintf(name, sizeof(name), "%s/%s", Com_Gamedir(), Cmd_Argv(1));

    // start the map up
    if (c > 2) {
      Cmd_ExecuteString(va("map %s", Cmd_Argv(2)), src_command);
      if (CLS_GetState() != ca_connected) return;
    }

    // open the demo file
    COM_AddExtension(name, ".dem", sizeof(name));

    Con_Printf("recording to %s.\n", name);
    cls.demofile = fopen(name, "wb");
    if (!cls.demofile) {
      Con_Printf("ERROR: couldn't create %s\n", name);
      return;
    }

    CLS_SetForceTrack(track);
    fprintf(cls.demofile, "%i\n", CLS_GetForceTrack());

    CLS_SetDemoRecording(true);

    // from ProQuake: initialize the demo file if we're already connected
    if (c == 2 && CLS_GetState() == ca_connected) {
      byte *data = net_message.daTa;
      int cursize = SB_GetCurSize(&net_message);
      int i;

      for (i = 0; i < 2; i++) {
        net_message.daTa = demo_head[i];
        SB_SetCurSize(&net_message, demo_head_size[i]);
        CL_WriteDemoMessage();
      }

      net_message.daTa = demo_head[2];
      SZ_Clear(&net_message);

      // current names, colors, and frag counts
      for (i = 0; i < CL_MaxClients(); i++) {
        MSG_WriteByte(&net_message, svc_updatename);
        MSG_WriteByte(&net_message, i);
        MSG_WriteString(&net_message, cl.scores[i].name);
        MSG_WriteByte(&net_message, svc_updatefrags);
        MSG_WriteByte(&net_message, i);
        MSG_WriteShort(&net_message, CL_ScoresFrags(i));
        MSG_WriteByte(&net_message, svc_updatecolors);
        MSG_WriteByte(&net_message, i);
        MSG_WriteByte(&net_message, CL_ScoresColors(i));
      }

      // send all current light styles
      for (i = 0; i < MAX_LIGHTSTYLES; i++) {
        MSG_WriteByte(&net_message, svc_lightstyle);
        MSG_WriteByte(&net_message, i);
        MSG_WriteString(&net_message, cl_lightstyle[i].map);
      }

      // what about the CD track or SVC fog... future consideration.
      MSG_WriteByte(&net_message, svc_updatestat);
      MSG_WriteByte(&net_message, STAT_TOTALSECRETS);
      MSG_WriteLong(&net_message, cl.stats[STAT_TOTALSECRETS]);

      MSG_WriteByte(&net_message, svc_updatestat);
      MSG_WriteByte(&net_message, STAT_TOTALMONSTERS);
      MSG_WriteLong(&net_message, cl.stats[STAT_TOTALMONSTERS]);

      MSG_WriteByte(&net_message, svc_updatestat);
      MSG_WriteByte(&net_message, STAT_SECRETS);
      MSG_WriteLong(&net_message, cl.stats[STAT_SECRETS]);

      MSG_WriteByte(&net_message, svc_updatestat);
      MSG_WriteByte(&net_message, STAT_MONSTERS);
      MSG_WriteLong(&net_message, cl.stats[STAT_MONSTERS]);

      // view entity
      MSG_WriteByte(&net_message, svc_setview);
      MSG_WriteShort(&net_message, CL_Viewentity());

      // signon
      MSG_WriteByte(&net_message, svc_signonnum);
      MSG_WriteByte(&net_message, 3);

      CL_WriteDemoMessage();

      // restore net_message
      net_message.daTa = data;
      SB_SetCurSize(&net_message, cursize);
    }
  */
}

/*
====================
CL_PlayDemo_f

play [demoname]
====================
*/
// public
void CL_PlayDemo_f(void) {
  /*
  char name[MAX_OSPATH];
  int i, c;
  qboolean neg;

  if (!IsSrcCommand()) return;

  if (Cmd_Argc() != 2) {
    Con_Printf("playdemo <demoname> : plays a demo\n");
    return;
  }

  // disconnect from server
  CL_Disconnect();

  // open the demo file
  q_strlcpy(name, Cmd_Argv(1), sizeof(name));
  COM_AddExtension(name, ".dem", sizeof(name));

  Con_Printf("Playing demo from %s.\n", name);

  COM_FOpenFile(name, &cls.demofile, NULL);
  if (!cls.demofile) {
    Con_Printf("ERROR: couldn't open %s\n", name);
    CLS_StopDemoCycle();  // stop demo loop
    return;
  }

  // ZOID, fscanf is evil
  // O.S.: if a space character e.g. 0x20 (' ') follows '\n',
  // fscanf skips that byte too and screws up further reads.
  //	fscanf (cls.demofile, "%i\n", &cls.forcetrack);
  CLS_SetForceTrack(0);
  c = 0; // silence pesky compiler warnings
  neg = false;
  // read a decimal integer possibly with a leading '-',
  // followed by a '\n':
  for (i = 0; i < 13; i++) {
    c = getc(cls.demofile);
    if (c == '\n') break;
    if (c == '-') {
      neg = true;
      continue;
    }
    // check for multiple '-' or legal digits? meh...
    CLS_SetForceTrack(CLS_GetForceTrack() * 10 + (c - '0'));
  }
  if (c != '\n') {
    fclose(cls.demofile);
    cls.demofile = NULL;
    CLS_StopDemoCycle();  // stop demo loop
    Con_Printf("ERROR: demo \"%s\" is invalid\n", name);
    return;
  }
  if (neg) CLS_SetForceTrack(-CLS_GetForceTrack());

  CLS_SetDemoPlayback(true);
  CLS_SetDemoPaused(false);
  CLS_SetState(ca_connected);

  // get rid of the menu and/or console
  SetKeyDest(key_game);
  */
}

/*
====================
CL_FinishTimeDemo

====================
*/
static void CL_FinishTimeDemo(void) {
  int frames;
  float time;

  CLS_SetTimeDemo(false);

  // the first frame didn't count
  frames = (host_framecount - CLS_GetTimeDemoStartFrame()) - 1;
  time = HostRealTime() - CLS_GetTimeDemoStartTime();
  if (!time) time = 1;
  Con_Printf("%i frames %5.1f seconds %5.1f fps\n", frames, time,
             frames / time);
}

/*
====================
CL_TimeDemo_f

timedemo [demoname]
====================
*/
// public
void CL_TimeDemo_f(void) {
  if (!IsSrcCommand()) return;

  if (Cmd_Argc() != 2) {
    Con_Printf("timedemo <demoname> : gets demo speeds\n");
    return;
  }

  CL_PlayDemo_f();
  if (!cls.demofile) return;

  // cls.td_starttime will be grabbed at the second frame of the demo, so
  // all the loading time doesn't get counted

  CLS_SetTimeDemo(true);
  CLS_SetTimeDemoStartFrame(host_framecount);
  CLS_SetTimeDemoLastFrame(-1);  // get a new message this frame
}
