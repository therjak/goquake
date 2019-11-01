#include "quakedef.h"

cvar_t scr_menuscale;
cvar_t scr_sbarscale;
cvar_t scr_sbaralpha;
cvar_t scr_crosshairscale;
cvar_t scr_showfps;
cvar_t scr_clock;

cvar_t scr_viewsize;
cvar_t scr_fov;
cvar_t scr_fov_adapt;
cvar_t gl_triplebuffer;

qboolean scr_initialized;  // ready to draw

int scr_tileclear_updates = 0;  // johnfitz

void ResetTileClearUpdates(void) { scr_tileclear_updates = 0; }

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

  ResetTileClearUpdates();

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

  SCR_SetVRect(r_refdef.vrect.x, r_refdef.vrect.y, r_refdef.vrect.width,
               r_refdef.vrect.height);
}

static void SCR_Callback_refdef(cvar_t *var) { SetRecalcRefdef(1); }

//============================================================================

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
  Cvar_FakeRegister(&gl_triplebuffer, "gl_triplebuffer");

  scr_initialized = true;
}

//============================================================================

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
    if (HostRealTime() - SCR_GetDisabledTime() > 60) {
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

  if (SCR_IsDrawDialog())  // new game confirm
  {
    if (Con_ForceDup())
      DrawConsoleBackgroundC();
    else
      Sbar_Draw();
    Draw_FadeScreen();
    SCR_DrawNotifyString();
  } else if (SCR_IsDrawLoading())  // loading
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
