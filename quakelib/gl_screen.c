// screen.c -- master for refresh, status bar, console, chat, notify, etc

#include "quakedef.h"

/*

background clear
rendering
turtle/net/ram icons
sbar
centerprint / slow centerprint
notify lines
intermission / finale overlay
loading plaque
console
menu

required background clears
required update regions


syncronous draw mode or async
One off screen buffer, with updates either copied or xblited
Need to double buffer?


async draw will require the refresh area to be cleared, because it will be
xblited, but sync draw can just ignore it.

sync
draw

CenterPrint ()
SlowPrint ()
Screen_Update ();
Con_Printf ();

net
turn off messages option

the refresh is allways rendered, unless the console is full screen


console is:
        notify lines
        half
        full

*/

float scr_con_current;
float scr_conlines;  // lines of console to display

cvar_t scr_menuscale;
cvar_t scr_sbarscale;
cvar_t scr_sbaralpha;
cvar_t scr_crosshairscale;
cvar_t scr_showfps;
cvar_t scr_clock;

cvar_t scr_viewsize;
cvar_t scr_fov;
cvar_t scr_fov_adapt;
cvar_t scr_conspeed;
cvar_t scr_centertime;
cvar_t scr_showram;
cvar_t scr_showturtle;
cvar_t scr_showpause;
cvar_t scr_printspeed;
cvar_t gl_triplebuffer;

extern cvar_t crosshair;

qboolean scr_initialized;  // ready to draw

qpic_t *scr_ram;
qpic_t *scr_net;
qpic_t *scr_turtle;

int clearconsole;

vrect_t scr_vrect;

qboolean scr_drawloading;
float scr_disabled_time;

int scr_tileclear_updates = 0;  // johnfitz

float GetScreenConsoleCurrentHeight(void) { return scr_con_current; }
/*
====================
AdaptFovx
Adapt a 4:3 horizontal FOV to the current screen size using the "Hor+" scaling:
2.0 * atan(width / height * 3.0 / 4.0 * tan(fov_x / 2.0))
====================
*/
float AdaptFovx(float fov_x, float width, float height) {
  float a, x;

  if (fov_x < 1 || fov_x > 179) Sys_Error("Bad fov: %f", fov_x);

  if (!Cvar_GetValue(&scr_fov_adapt)) return fov_x;
  if ((x = height / width) == 0.75) return fov_x;
  a = atan(0.75 / x * tan(fov_x / 360 * M_PI));
  a = a * 360 / M_PI;
  return a;
}

/*
====================
CalcFovy
====================
*/
float CalcFovy(float fov_x, float width, float height) {
  float a, x;

  if (fov_x < 1 || fov_x > 179) Sys_Error("Bad fov: %f", fov_x);

  x = width / tan(fov_x / 360 * M_PI);
  a = atan(height / x);
  a = a * 360 / M_PI;
  return a;
}

/*
=================
SCR_CalcRefdef

Must be called whenever vid changes
Internal use only
=================
*/
static void SCR_CalcRefdef(void) {
  float size;

  // force the status bar to redraw
  Sbar_Changed();

  scr_tileclear_updates = 0;  // johnfitz

  // bound viewsize
  if (Cvar_GetValue(&scr_viewsize) < 30) Cvar_SetQuick(&scr_viewsize, "30");
  if (Cvar_GetValue(&scr_viewsize) > 120) Cvar_SetQuick(&scr_viewsize, "120");

  // bound fov
  if (Cvar_GetValue(&scr_fov) < 10) Cvar_SetQuick(&scr_fov, "10");
  if (Cvar_GetValue(&scr_fov) > 170) Cvar_SetQuick(&scr_fov, "170");

  SetRecalcRefdef(0);

  size = q_min(Cvar_GetValue(&scr_viewsize), 100) / 100;

  r_refdef.vrect.width =
      q_max(GL_Width() * size, 96);  // no smaller than 96, for icons
  r_refdef.vrect.height = q_min(
      GL_Height() * size, GL_Height() - Sbar_Lines());  // make room for sbar
  r_refdef.vrect.x = (GL_Width() - r_refdef.vrect.width) / 2;
  r_refdef.vrect.y = (GL_Height() - Sbar_Lines() - r_refdef.vrect.height) / 2;

  r_refdef.fov_x =
      AdaptFovx(Cvar_GetValue(&scr_fov), ScreenWidth(), ScreenHeight());
  r_refdef.fov_y =
      CalcFovy(r_refdef.fov_x, r_refdef.vrect.width, r_refdef.vrect.height);

  scr_vrect = r_refdef.vrect;
}

static void SCR_Callback_refdef(cvar_t *var) { SetRecalcRefdef(1); }

//============================================================================

/*
==================
SCR_LoadPics -- johnfitz
==================
*/
void SCR_LoadPics(void) {
  scr_ram = Draw_PicFromWad("ram");
  scr_net = Draw_PicFromWad("net");
  scr_turtle = Draw_PicFromWad("turtle");
}

/*
==================
SCR_Init
==================
*/
void SCR_Init(void) {
  Cvar_FakeRegister(&scr_menuscale, "scr_menuscale");
  Cvar_FakeRegister(&scr_sbarscale, "scr_sbarscale");
  Cvar_SetCallback(&scr_sbaralpha, SCR_Callback_refdef);
  Cvar_FakeRegister(&scr_sbaralpha, "scr_sbaralpha");
  Cvar_FakeRegister(&scr_crosshairscale, "scr_crosshairscale");
  Cvar_FakeRegister(&scr_showfps, "scr_showfps");
  Cvar_FakeRegister(&scr_clock, "scr_clock");
  Cvar_SetCallback(&scr_fov, SCR_Callback_refdef);
  Cvar_SetCallback(&scr_fov_adapt, SCR_Callback_refdef);
  Cvar_SetCallback(&scr_viewsize, SCR_Callback_refdef);
  Cvar_FakeRegister(&scr_fov, "fov");
  Cvar_FakeRegister(&scr_fov_adapt, "fov_adapt");
  Cvar_FakeRegister(&scr_viewsize, "viewsize");
  Cvar_FakeRegister(&scr_conspeed, "scr_conspeed");
  Cvar_FakeRegister(&scr_showram, "showram");
  Cvar_FakeRegister(&scr_showturtle, "showturtle");
  Cvar_FakeRegister(&scr_showpause, "showpause");
  Cvar_FakeRegister(&scr_centertime, "scr_centertime");
  Cvar_FakeRegister(&scr_printspeed, "scr_printspeed");
  Cvar_FakeRegister(&gl_triplebuffer, "gl_triplebuffer");

  SCR_LoadPics();  // johnfitz

  scr_initialized = true;
}

//============================================================================

/*
==============
SCR_DrawFPS -- johnfitz
==============
*/
void SCR_DrawFPS(void) {
  static double oldtime = 0;
  static double lastfps = 0;
  static int oldframecount = 0;
  double elapsed_time;
  int frames;

  elapsed_time = HostRealTime() - oldtime;
  frames = r_framecount - oldframecount;

  if (elapsed_time < 0 || frames < 0) {
    oldtime = HostRealTime();
    oldframecount = r_framecount;
    return;
  }
  // update value every 3/4 second
  if (elapsed_time > 0.75) {
    lastfps = frames / elapsed_time;
    oldtime = HostRealTime();
    oldframecount = r_framecount;
  }

  if (Cvar_GetValue(&scr_showfps)) {
    char st[16];
    int x, y;
    sprintf(st, "%4.0f fps", lastfps);
    x = 320 - (strlen(st) << 3);
    y = 200 - 8;
    if (Cvar_GetValue(&scr_clock)) y -= 8;  // make room for clock
    GL_SetCanvas(CANVAS_BOTTOMRIGHT);
    Draw_String(x, y, st);
    scr_tileclear_updates = 0;
  }
}

/*
==============
SCR_DrawClock -- johnfitz
==============
*/
void SCR_DrawClock(void) {
  char str[12];

  if (Cvar_GetValue(&scr_clock) == 1) {
    int minutes, seconds;

    minutes = CL_Time() / 60;
    seconds = ((int)CL_Time()) % 60;

    sprintf(str, "%i:%i%i", minutes, seconds / 10, seconds % 10);
  } else
    return;

  // draw it
  GL_SetCanvas(CANVAS_BOTTOMRIGHT);
  Draw_String(320 - (strlen(str) << 3), 200 - 8, str);

  scr_tileclear_updates = 0;
}

/*
==============
SCR_DrawDevStats
==============
*/
void SCR_DrawDevStats(void) {
  char str[40];
  int y = 25 - 9;  // 9=number of lines to print
  int x = 0;       // margin

  if (!Cvar_GetValue(&devstats)) return;

  GL_SetCanvas(CANVAS_BOTTOMLEFT);

  DrawFillC(x, y * 8, 19 * 8, 9 * 8, 0, 0.5);  // dark rectangle

  sprintf(str, "devstats |Curr Peak");
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "---------+---------");
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Edicts   |%4i %4i", dev_stats.edicts, dev_peakstats.edicts);
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Packet   |%4i %4i", dev_stats.packetsize,
          dev_peakstats.packetsize);
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Visedicts|%4i %4i", dev_stats.visedicts,
          dev_peakstats.visedicts);
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Efrags   |%4i %4i", dev_stats.efrags, dev_peakstats.efrags);
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Dlights  |%4i %4i", dev_stats.dlights, dev_peakstats.dlights);
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Beams    |%4i %4i", dev_stats.beams, dev_peakstats.beams);
  Draw_String(x, (y++) * 8 - x, str);

  sprintf(str, "Tempents |%4i %4i", dev_stats.tempents, dev_peakstats.tempents);
  Draw_String(x, (y++) * 8 - x, str);
}

/*
==============
SCR_DrawRam
==============
*/
void SCR_DrawRam(void) {
  if (!Cvar_GetValue(&scr_showram)) return;

  if (!r_cache_thrash) return;

  GL_SetCanvas(CANVAS_DEFAULT);  // johnfitz

  Draw_Pic(scr_vrect.x + 32, scr_vrect.y, scr_ram);
}

/*
==============
SCR_DrawTurtle
==============
*/
void SCR_DrawTurtle(void) {
  static int count;

  if (!Cvar_GetValue(&scr_showturtle)) return;

  if (HostFrameTime() < 0.1) {
    count = 0;
    return;
  }

  count++;
  if (count < 3) return;

  GL_SetCanvas(CANVAS_DEFAULT);  // johnfitz

  Draw_Pic(scr_vrect.x, scr_vrect.y, scr_turtle);
}

/*
==============
SCR_DrawNet
==============
*/
void SCR_DrawNet(void) {
  if (HostRealTime() - CL_LastReceivedMessage() < 0.3) return;
  if (CLS_IsDemoPlayback()) return;

  GL_SetCanvas(CANVAS_DEFAULT);  // johnfitz

  Draw_Pic(scr_vrect.x + 64, scr_vrect.y, scr_net);
}

/*
==============
DrawPause
==============
*/
void SCR_DrawPause(void) {
  qpic_t *pic;

  if (!CL_Paused()) return;

  if (!Cvar_GetValue(&scr_showpause))  // turn off for screenshots
    return;

  GL_SetCanvas(CANVAS_MENU);  // johnfitz

  pic = Draw_CachePic("gfx/pause.lmp");
  Draw_Pic((320 - pic->width) / 2, (240 - 48 - pic->height) / 2,
           pic);  // johnfitz -- stretched menus

  scr_tileclear_updates = 0;  // johnfitz
}

/*
==============
SCR_DrawLoading
==============
*/
void SCR_DrawLoading(void) {
  qpic_t *pic;

  if (!scr_drawloading) return;

  GL_SetCanvas(CANVAS_MENU);  // johnfitz

  pic = Draw_CachePic("gfx/loading.lmp");
  Draw_Pic((320 - pic->width) / 2, (240 - 48 - pic->height) / 2,
           pic);  // johnfitz -- stretched menus

  scr_tileclear_updates = 0;  // johnfitz
}

/*
==============
SCR_DrawCrosshair -- johnfitz
==============
*/
void SCR_DrawCrosshair(void) {
  if (!Cvar_GetValue(&crosshair)) return;

  GL_SetCanvas(CANVAS_CROSSHAIR);
  Draw_Character(-4, -4, '+');  // 0,0 is center of viewport
}

//=============================================================================

/*
==================
SCR_SetUpToDrawConsole
==================
*/
void SCR_SetUpToDrawConsole(void) {
  // johnfitz -- let's hack away the problem of slow console when host_timescale
  // is <0
  extern cvar_t host_timescale;
  float timescale;
  // johnfitz

  Con_CheckResize();

  if (scr_drawloading) return;  // never a console with loading plaque

  // decide on the height of the console
  Con_SetForceDup(!cl.worldmodel || CLS_GetSignon() != SIGNONS);

  if (Con_ForceDup()) {
    scr_conlines = GL_Height();
    scr_con_current = scr_conlines;
  } else if (GetKeyDest() == key_console)
    scr_conlines = GL_Height() / 2;
  else
    scr_conlines = 0;  // none visible

  timescale =
      (Cvar_GetValue(&host_timescale) > 0) ? Cvar_GetValue(&host_timescale) : 1;

  if (scr_conlines < scr_con_current) {
    // ericw -- (GL_Height()/600.0) factor makes conspeed resolution
    // independent, using 800x600 as a baseline
    scr_con_current -= Cvar_GetValue(&scr_conspeed) * (GL_Height() / 600.0) *
                       HostFrameTime() / timescale;
    if (scr_conlines > scr_con_current) scr_con_current = scr_conlines;
  } else if (scr_conlines > scr_con_current) {
    // ericw -- (GL_Height()/600.0)
    scr_con_current += Cvar_GetValue(&scr_conspeed) * (GL_Height() / 600.0) *
                       HostFrameTime() / timescale;
    if (scr_conlines < scr_con_current) scr_con_current = scr_conlines;
  }

  if (clearconsole++ < GetNumPages()) Sbar_Changed();

  if (!Con_ForceDup() && scr_con_current)
    scr_tileclear_updates = 0;  // johnfitz
}

/*
==================
SCR_DrawConsole
==================
*/
void SCR_DrawConsole(void) {
  if (scr_con_current) {
    Con_DrawConsole(scr_con_current);
    clearconsole = 0;
  } else {
    if (GetKeyDest() == key_game || GetKeyDest() == key_message)
      Con_DrawNotify();  // only draw notify in game
  }
}

/*
===============
SCR_BeginLoadingPlaque

================
*/
void SCR_BeginLoadingPlaque(void) {
  S_StopAllSounds(true);

  if (CLS_GetState() != ca_connected) return;
  if (CLS_GetSignon() != SIGNONS) return;

  // redraw with no console and the loading plaque
  Con_ClearNotify();
  scr_con_current = 0;

  scr_drawloading = true;
  Sbar_Changed();
  SCR_UpdateScreen();
  scr_drawloading = false;

  SetScreenDisabled(true);
  scr_disabled_time = HostRealTime();
}

//=============================================================================

const char *scr_notifystring;
qboolean scr_drawdialog;

void SCR_DrawNotifyString(void) {
  const char *start;
  int l;
  int j;
  int x, y;

  GL_SetCanvas(CANVAS_MENU);  // johnfitz

  start = scr_notifystring;

  y = 200 * 0.35;  // johnfitz -- stretched overlays

  do {
    // scan the width of the line
    for (l = 0; l < 40; l++)
      if (start[l] == '\n' || !start[l]) break;
    x = (320 - l * 8) / 2;  // johnfitz -- stretched overlays
    for (j = 0; j < l; j++, x += 8) Draw_Character(x, y, start[j]);

    y += 8;

    while (*start && *start != '\n') start++;

    if (!*start) break;
    start++;  // skip the \n
  } while (1);
}

/*
==================
SCR_ModalMessage

Displays a text string in the center of the screen and waits for a Y or N
keypress.
==================
*/
int SCR_ModalMessage(const char *text, float timeout)  // johnfitz -- timeout
{
  double time1, time2;  // johnfitz -- timeout
  int lastkey, lastchar;

  if (CLS_GetState() == ca_dedicated) return true;

  scr_notifystring = text;

  // draw a fresh screen
  scr_drawdialog = true;
  SCR_UpdateScreen();
  scr_drawdialog = false;

  S_ClearBuffer();  // so dma doesn't loop current sound

  return KeyModalResult(timeout);
}

//=============================================================================

// johnfitz -- deleted SCR_BringDownConsole

/*
==================
SCR_TileClear
==================
*/
void SCR_TileClear(void) {
  // ericw -- added check for glsl gamma. TODO: remove this ugly optimization?
  if (scr_tileclear_updates >= GetNumPages() && !Cvar_GetValue(&gl_clear) &&
      !(Cvar_GetValue(&vid_gamma) != 1))
    return;
  scr_tileclear_updates++;

  if (r_refdef.vrect.x > 0) {
    // left
    Draw_TileClear(0, 0, r_refdef.vrect.x, GL_Height() - Sbar_Lines());
    // right
    Draw_TileClear(r_refdef.vrect.x + r_refdef.vrect.width, 0,
                   GL_Width() - r_refdef.vrect.x - r_refdef.vrect.width,
                   GL_Height() - Sbar_Lines());
  }

  if (r_refdef.vrect.y > 0) {
    // top
    Draw_TileClear(r_refdef.vrect.x, 0, r_refdef.vrect.width, r_refdef.vrect.y);
    // bottom
    Draw_TileClear(
        r_refdef.vrect.x, r_refdef.vrect.y + r_refdef.vrect.height,
        r_refdef.vrect.width,
        GL_Height() - r_refdef.vrect.y - r_refdef.vrect.height - Sbar_Lines());
  }
}

/*
==================
SCR_UpdateScreen

This is called every frame, and can also be called explicitly to flush
text to the screen.

WARNING: be very careful calling this from elsewhere, because the refresh
needs almost the entire 256k of stack space!
==================
*/
void SCR_UpdateScreen(void) {
  SetNumPages((Cvar_GetValue(&gl_triplebuffer)) ? 3 : 2);

  if (ScreenDisabled()) {
    if (HostRealTime() - scr_disabled_time > 60) {
      SetScreenDisabled(false);
      Con_Printf("load failed.\n");
    } else
      return;
  }

  if (!scr_initialized || !Con_Initialized()) return;  // not initialized yet

  UpdateViewport();

  //
  // determine size of refresh window
  //
  if (GetRecalcRefdef()) SCR_CalcRefdef();

  //
  // do 3D refresh drawing, and then update the screen
  //
  SCR_SetUpToDrawConsole();

  V_RenderView();

  GL_Set2D();

  // FIXME: only call this when needed
  SCR_TileClear();

  if (scr_drawdialog)  // new game confirm
  {
    if (Con_ForceDup())
      DrawConsoleBackgroundC();
    else
      Sbar_Draw();
    Draw_FadeScreen();
    SCR_DrawNotifyString();
  } else if (scr_drawloading)  // loading
  {
    SCR_DrawLoading();
    Sbar_Draw();
  } else if (CL_Intermission() == 1 &&
             GetKeyDest() == key_game)  // end of level
  {
    Sbar_IntermissionOverlay();
  } else if (CL_Intermission() == 2 &&
             GetKeyDest() == key_game)  // end of episode
  {
    Sbar_FinaleOverlay();
    SCR_CheckDrawCenterString();
  } else {
    SCR_DrawCrosshair();  // johnfitz
    SCR_DrawRam();
    SCR_DrawNet();
    SCR_DrawTurtle();
    SCR_DrawPause();
    SCR_CheckDrawCenterString();
    Sbar_Draw();
    SCR_DrawDevStats();  // johnfitz
    SCR_DrawFPS();       // johnfitz
    SCR_DrawClock();     // johnfitz
    SCR_DrawConsole();
    M_Draw();
  }

  V_UpdateBlend();  // johnfitz -- V_UpdatePalette cleaned up and renamed

  GLSLGamma_GammaCorrect();

  GL_EndRendering();
}
