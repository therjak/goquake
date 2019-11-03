#include "quakedef.h"

cvar_t scr_menuscale;
cvar_t scr_sbarscale;
cvar_t scr_crosshairscale;

void SetRefdefRect(int x,int y,int w,int h) {
  r_refdef.vrect.x = x;
  r_refdef.vrect.y = y;
  r_refdef.vrect.width = w;
  r_refdef.vrect.height = h;
}

void SetRefdefFov(float x,float y) {
  r_refdef.fov_x = x;
  r_refdef.fov_y = y;
}

void SCR_Init(void) {
  Cvar_FakeRegister(&scr_menuscale, "scr_menuscale");
  Cvar_FakeRegister(&scr_sbarscale, "scr_sbarscale");
  Cvar_FakeRegister(&scr_crosshairscale, "scr_crosshairscale");

  SCR_InitGo(); // just prevents drawing to early
}

