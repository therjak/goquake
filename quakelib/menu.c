#include "quakedef.h"

void M_Menu_GameOptions_f(void);
void M_Menu_Search_f(void);
void M_Menu_ServerList_f(void);

void M_GameOptions_Draw(void);
void M_Search_Draw(void);
void M_ServerList_Draw(void);

void M_GameOptions_Key(int key);
void M_Search_Key(int key);
void M_ServerList_Key(int key);

qboolean m_return_onerror;
char m_return_reason[32];


void M_ConfigureNetSubsystem(void);

/*
================
M_DrawCharacter

Draws one solid graphics character
================
*/
void M_DrawCharacter(int cx, int line, int num) {
  Draw_Character(cx, line, num);
}

void M_Print(int cx, int cy, const char *str) {
  while (*str) {
    M_DrawCharacter(cx, cy, (*str) + 128);
    str++;
    cx += 8;
  }
}

void M_PrintWhite(int cx, int cy, const char *str) {
  while (*str) {
    M_DrawCharacter(cx, cy, *str);
    str++;
    cx += 8;
  }
}

void M_DrawTransPic(int x, int y, qpic_t *pic) { Draw_Pic(x, y, pic); }

void M_DrawTransPicTranslate(int x, int y, qpic_t *pic, int top,
                             int bottom)  // johnfitz -- more parameters
{
  Draw_TransPicTranslate(x, y, pic, top, bottom);
}

void M_DrawTextBox(int x, int y, int width, int lines) {
  qpic_t *p;
  int cx, cy;
  int n;

  // draw left side
  cx = x;
  cy = y;
  p = Draw_CachePic("gfx/box_tl.lmp");
  M_DrawTransPic(cx, cy, p);
  p = Draw_CachePic("gfx/box_ml.lmp");
  for (n = 0; n < lines; n++) {
    cy += 8;
    M_DrawTransPic(cx, cy, p);
  }
  p = Draw_CachePic("gfx/box_bl.lmp");
  M_DrawTransPic(cx, cy + 8, p);

  // draw middle
  cx += 8;
  while (width > 0) {
    cy = y;
    p = Draw_CachePic("gfx/box_tm.lmp");
    M_DrawTransPic(cx, cy, p);
    p = Draw_CachePic("gfx/box_mm.lmp");
    for (n = 0; n < lines; n++) {
      cy += 8;
      if (n == 1) p = Draw_CachePic("gfx/box_mm2.lmp");
      M_DrawTransPic(cx, cy, p);
    }
    p = Draw_CachePic("gfx/box_bm.lmp");
    M_DrawTransPic(cx, cy + 8, p);
    width -= 2;
    cx += 16;
  }

  // draw right side
  cy = y;
  p = Draw_CachePic("gfx/box_tr.lmp");
  M_DrawTransPic(cx, cy, p);
  p = Draw_CachePic("gfx/box_mr.lmp");
  for (n = 0; n < lines; n++) {
    cy += 8;
    M_DrawTransPic(cx, cy, p);
  }
  p = Draw_CachePic("gfx/box_br.lmp");
  M_DrawTransPic(cx, cy + 8, p);
}

//=============================================================================
/* SEARCH MENU */

qboolean searchComplete = false;
double searchCompleteTime;

void M_Menu_Search_f(void) {
  IN_Deactivate();
  SetKeyDest(key_menu);
  MENU_SetState(m_search);
  MENU_SetEnterSound(false);
  slistSilent = true;
  slistLocal = false;
  searchComplete = false;
  NET_Slist_f();
}

void M_Search_Draw(void) {
  qpic_t *p;
  int x;

  p = Draw_CachePic("gfx/p_multi.lmp");
  Draw_Pic((320 - p->width) / 2, 4, p);
  x = (320 / 2) - ((12 * 8) / 2) + 4;
  M_DrawTextBox(x - 8, 32, 12, 1);
  M_Print(x, 40, "Searching...");

  if (slistInProgress) {
    NET_Poll();
    return;
  }

  if (!searchComplete) {
    searchComplete = true;
    searchCompleteTime = HostRealTime();
  }

  if (hostCacheCount) {
    M_Menu_ServerList_f();
    return;
  }

  M_PrintWhite((320 / 2) - ((22 * 8) / 2), 64, "No Quake servers found");
  if ((HostRealTime() - searchCompleteTime) < 3.0) return;

  M_Menu_LanConfig_f();
}

void M_Search_Key(int key) {}

//=============================================================================
/* SLIST MENU */

int slist_cursor;
qboolean slist_sorted;

void M_Menu_ServerList_f(void) {
  IN_Deactivate();
  SetKeyDest(key_menu);
  MENU_SetState(m_slist);
  MENU_SetEnterSound(true);
  slist_cursor = 0;
  m_return_onerror = false;
  m_return_reason[0] = 0;
  slist_sorted = false;
}

void M_ServerList_Draw(void) {
  int n;
  qpic_t *p;

  if (!slist_sorted) {
    slist_sorted = true;
    NET_SlistSort();
  }

  p = Draw_CachePic("gfx/p_multi.lmp");
  Draw_Pic((320 - p->width) / 2, 4, p);
  for (n = 0; n < hostCacheCount; n++)
    M_Print(16, 32 + 8 * n, NET_SlistPrintServer(n));
  M_DrawCharacter(0, 32 + slist_cursor * 8,
                  12 + ((int)(HostRealTime() * 4) & 1));

  if (*m_return_reason) M_PrintWhite(16, 148, m_return_reason);
}

void M_ServerList_Key(int k) {
  switch (k) {
    case K_ESCAPE:
    case K_BBUTTON:
      M_Menu_LanConfig_f();
      break;

    case K_SPACE:
      M_Menu_Search_f();
      break;

    case K_UPARROW:
    case K_LEFTARROW:
      S_LocalSound("misc/menu1.wav");
      slist_cursor--;
      if (slist_cursor < 0) slist_cursor = hostCacheCount - 1;
      break;

    case K_DOWNARROW:
    case K_RIGHTARROW:
      S_LocalSound("misc/menu1.wav");
      slist_cursor++;
      if (slist_cursor >= hostCacheCount) slist_cursor = 0;
      break;

    case K_ENTER:
    case K_KP_ENTER:
    case K_ABUTTON:
      S_LocalSound("misc/menu2.wav");
      m_return_onerror = true;
      slist_sorted = false;
      IN_Activate();
      SetKeyDest(key_game);
      MENU_SetState(m_none);
      Cbuf_AddText(
          va("connect \"%s\"\n", NET_SlistPrintServerName(slist_cursor)));
      break;

    default:
      break;
  }
}

//=============================================================================
/* Menu Subsystem */
void M_ConfigureNetSubsystem(void) {
  // enable/disable net systems to match desired config
  Cbuf_AddText("stopdemo\n");

  //therjak: this should happen
  //net_hostport = lanConfig_port;
}
