#include "quakedef.h"

#define NET_MAXMESSAGE 64000

// THERJAK: this should be possible to move now

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

void CL_WriteDemoMessage(void) {
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

