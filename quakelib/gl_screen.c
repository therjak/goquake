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

int glx, gly, glwidth, glheight;

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
int clearnotify;

vrect_t scr_vrect;

qboolean scr_drawloading;
float scr_disabled_time;

int scr_tileclear_updates = 0;  // johnfitz

void SCR_ScreenShot_f(void);

float GetScreenConsoleCurrentHeight(void) { return scr_con_current; }
/*
===============================================================================

CENTER PRINTING

===============================================================================
*/

char scr_centerstring[1024];
float scr_centertime_start;  // for slow victory printing
float scr_centertime_off;
int scr_center_lines;
int scr_erase_lines;
int scr_erase_center;

/*
==============
SCR_CenterPrint

Called for important messages that should stay in the center of the screen
for a few moments
==============
*/
void SCR_CenterPrint(const char *str)  // update centerprint data
{
  strncpy(scr_centerstring, str, sizeof(scr_centerstring) - 1);
  scr_centertime_off = Cvar_GetValue(&scr_centertime);
  scr_centertime_start = CL_Time();

  // count the number of lines for centering
  scr_center_lines = 1;
  str = scr_centerstring;
  while (*str) {
    if (*str == '\n') scr_center_lines++;
    str++;
  }
}

void SCR_DrawCenterString(void)  // actually do the drawing
{
  char *start;
  int l;
  int j;
  int x, y;
  int remaining;

  GL_SetCanvas(CANVAS_MENU);  // johnfitz

  // the finale prints the characters one at a time
  if (CL_Intermission())
    remaining =
        Cvar_GetValue(&scr_printspeed) * (CL_Time() - scr_centertime_start);
  else
    remaining = 9999;

  scr_erase_center = 0;
  start = scr_centerstring;

  if (scr_center_lines <= 4)
    y = 200 * 0.35;  // johnfitz -- 320x200 coordinate system
  else
    y = 48;
  if (Cvar_GetValue(&crosshair)) y -= 8;

  do {
    // scan the width of the line
    for (l = 0; l < 40; l++)
      if (start[l] == '\n' || !start[l]) break;
    x = (320 - l * 8) / 2;  // johnfitz -- 320x200 coordinate system
    for (j = 0; j < l; j++, x += 8) {
      Draw_Character(x, y, start[j]);  // johnfitz -- stretch overlays
      if (!remaining--) return;
    }

    y += 8;

    while (*start && *start != '\n') start++;

    if (!*start) break;
    start++;  // skip the \n
  } while (1);
}

void SCR_CheckDrawCenterString(void) {
  if (scr_center_lines > scr_erase_lines) scr_erase_lines = scr_center_lines;

  scr_centertime_off -= HostFrameTime();

  if (scr_centertime_off <= 0 && !CL_Intermission()) return;
  if (GetKeyDest() != key_game) return;
  if (CL_Paused())  // johnfitz -- don't show centerprint during a pause
    return;

  SCR_DrawCenterString();
}

//=============================================================================

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
  float size, scale;  // johnfitz -- scale

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

  // johnfitz -- rewrote this section
  size = Cvar_GetValue(&scr_viewsize);
  scale = CLAMP(1.0, Cvar_GetValue(&scr_sbarscale), (float)glwidth / 320.0);

  // johnfitz -- scr_sbaralpha.value
  if (size >= 120 || CL_Intermission() || Cvar_GetValue(&scr_sbaralpha) < 1) {
    sb_lines = 0;
  } else if (size >= 110) {
    sb_lines = 24 * scale;
  } else {
    sb_lines = 48 * scale;
  }

  size = q_min(Cvar_GetValue(&scr_viewsize), 100) / 100;
  // johnfitz

  // johnfitz -- rewrote this section
  r_refdef.vrect.width =
      q_max(glwidth * size, 96);  // no smaller than 96, for icons
  r_refdef.vrect.height =
      q_min(glheight * size, glheight - sb_lines);  // make room for sbar
  r_refdef.vrect.x = (glwidth - r_refdef.vrect.width) / 2;
  r_refdef.vrect.y = (glheight - sb_lines - r_refdef.vrect.height) / 2;
  // johnfitz

  r_refdef.fov_x =
      AdaptFovx(Cvar_GetValue(&scr_fov), ScreenWidth(), ScreenHeight());
  r_refdef.fov_y =
      CalcFovy(r_refdef.fov_x, r_refdef.vrect.width, r_refdef.vrect.height);

  scr_vrect = r_refdef.vrect;
}

/*
=================
SCR_SizeUp_f

Keybinding command
=================
*/
void SCR_SizeUp_f(void) {
  Cvar_SetValueQuick(&scr_viewsize, Cvar_GetValue(&scr_viewsize) + 10);
}

/*
=================
SCR_SizeDown_f

Keybinding command
=================
*/
void SCR_SizeDown_f(void) {
  Cvar_SetValueQuick(&scr_viewsize, Cvar_GetValue(&scr_viewsize) - 10);
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

  Cmd_AddCommand("screenshot", SCR_ScreenShot_f);
  Cmd_AddCommand("sizeup", SCR_SizeUp_f);
  Cmd_AddCommand("sizedown", SCR_SizeDown_f);

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

  Draw_Fill(x, y * 8, 19 * 8, 9 * 8, 0, 0.5);  // dark rectangle

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
  con_forcedup = !cl.worldmodel || CLS_GetSignon() != SIGNONS;

  if (con_forcedup) {
    scr_conlines = glheight;
    scr_con_current = scr_conlines;
  } else if (GetKeyDest() == key_console)
    scr_conlines = glheight / 2;
  else
    scr_conlines = 0;  // none visible

  timescale =
      (Cvar_GetValue(&host_timescale) > 0) ? Cvar_GetValue(&host_timescale) : 1;

  if (scr_conlines < scr_con_current) {
    // ericw -- (glheight/600.0) factor makes conspeed resolution independent,
    // using 800x600 as a baseline
    scr_con_current -= Cvar_GetValue(&scr_conspeed) * (glheight / 600.0) *
                       HostFrameTime() / timescale;
    if (scr_conlines > scr_con_current) scr_con_current = scr_conlines;
  } else if (scr_conlines > scr_con_current) {
    // ericw -- (glheight/600.0)
    scr_con_current += Cvar_GetValue(&scr_conspeed) * (glheight / 600.0) *
                       HostFrameTime() / timescale;
    if (scr_conlines < scr_con_current) scr_con_current = scr_conlines;
  }

  if (clearconsole++ < GetNumPages()) Sbar_Changed();

  if (!con_forcedup && scr_con_current) scr_tileclear_updates = 0;  // johnfitz
}

/*
==================
SCR_DrawConsole
==================
*/
void SCR_DrawConsole(void) {
  if (scr_con_current) {
    Con_DrawConsole(scr_con_current, true);
    clearconsole = 0;
  } else {
    if (GetKeyDest() == key_game || GetKeyDest() == key_message)
      Con_DrawNotify();  // only draw notify in game
  }
}

/*
==============================================================================

SCREEN SHOTS

==============================================================================
*/

/*
==================
SCR_ScreenShot_f -- johnfitz -- rewritten to use Image_WriteTGA
==================
*/
void SCR_ScreenShot_f(void) {
  byte *buffer;
  char pngname[16];  // johnfitz -- was [80]
  char checkname[MAX_OSPATH];
  int i;

  // find a file name to save it to
  for (i = 0; i < 10000; i++) {
    q_snprintf(pngname, sizeof(pngname), "spasm%04i.png", i);  // "fitz%04i.tga"
    q_snprintf(checkname, sizeof(checkname), "%s/%s", Com_Gamedir(), pngname);
    if (Sys_FileTime(checkname) == -1) break;  // file doesn't exist
  }
  if (i == 10000) {
    Con_Printf("SCR_ScreenShot_f: Couldn't find an unused filename\n");
    return;
  }

  // get data
  if (!(buffer = (byte *)malloc(glwidth * glheight * 4))) {
    Con_Printf("SCR_ScreenShot_f: Couldn't allocate memory\n");
    return;
  }

  glPixelStorei(GL_PACK_ALIGNMENT,
                1); /* for widths that aren't a multiple of 4 */
  glReadPixels(glx, gly, glwidth, glheight, GL_RGBA, GL_UNSIGNED_BYTE, buffer);

  // now write the file
  if (Image_Write(checkname, buffer, glwidth, glheight))
    Con_Printf("Wrote %s\n", pngname);
  else
    Con_Printf("SCR_ScreenShot_f: Couldn't create a TGA file\n");

  free(buffer);
}

//=============================================================================

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
  scr_centertime_off = 0;
  scr_con_current = 0;

  scr_drawloading = true;
  Sbar_Changed();
  SCR_UpdateScreen();
  scr_drawloading = false;

  SetScreenDisabled(true);
  scr_disabled_time = HostRealTime();
}

/*
===============
SCR_EndLoadingPlaque

================
*/
void SCR_EndLoadingPlaque(void) {
  SetScreenDisabled(false);
  Con_ClearNotify();
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

  time1 = Sys_DoubleTime() + timeout;  // johnfitz -- timeout
  time2 = 0.0f;                        // johnfitz -- timeout

  Key_BeginInputGrab();
  do {
    IN_SendKeyEvents();
    Key_GetGrabbedInput(&lastkey, &lastchar);
    Sys_Sleep(16);
    if (timeout)
      time2 = Sys_DoubleTime();  // johnfitz -- zero timeout means wait forever.
  } while (lastchar != 'y' && lastchar != 'Y' && lastchar != 'n' &&
           lastchar != 'N' && lastkey != K_ESCAPE && lastkey != K_ABUTTON &&
           lastkey != K_BBUTTON && time2 <= time1);
  Key_EndInputGrab();

  //	SCR_UpdateScreen (); //johnfitz -- commented out

  // johnfitz -- timeout
  if (time2 > time1) return false;
  // johnfitz

  return (lastchar == 'y' || lastchar == 'Y' || lastkey == K_ABUTTON);
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
    Draw_TileClear(0, 0, r_refdef.vrect.x, glheight - sb_lines);
    // right
    Draw_TileClear(r_refdef.vrect.x + r_refdef.vrect.width, 0,
                   glwidth - r_refdef.vrect.x - r_refdef.vrect.width,
                   glheight - sb_lines);
  }

  if (r_refdef.vrect.y > 0) {
    // top
    Draw_TileClear(r_refdef.vrect.x, 0, r_refdef.vrect.width, r_refdef.vrect.y);
    // bottom
    Draw_TileClear(
        r_refdef.vrect.x, r_refdef.vrect.y + r_refdef.vrect.height,
        r_refdef.vrect.width,
        glheight - r_refdef.vrect.y - r_refdef.vrect.height - sb_lines);
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

  if (!scr_initialized || !con_initialized) return;  // not initialized yet

  glx = 0;
  gly = 0;
  glwidth = ScreenWidth();
  glheight = ScreenHeight();

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
    if (con_forcedup)
      Draw_ConsoleBackground();
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
