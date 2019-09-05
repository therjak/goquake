// sbar.c -- status bar code

#include "_cgo_export.h"
//
#include "quakedef.h"

qpic_t *sb_sbar;
qpic_t *sb_scorebar;

qpic_t *sb_items[32];

qpic_t *rsb_teambord;  // PGM 01/19/97 - team color border

/*
===============
Sbar_LoadPics -- johnfitz -- load all the sbar pics
===============
*/
void Sbar_LoadPicsC(void) {
  sb_items[0] = Draw_PicFromWad("sb_key1");
  sb_items[1] = Draw_PicFromWad("sb_key2");

  sb_sbar = Draw_PicFromWad("sbar");
  sb_scorebar = Draw_PicFromWad("scorebar");

  if (CMLRogue()) {
    rsb_teambord = Draw_PicFromWad("r_teambord");
  }
}

//=============================================================================

int fragsort[MAX_SCOREBOARD];
int scoreboardlines;

/*
===============
Sbar_SortFrags
===============
*/
void Sbar_SortFrags(void) {
  int i, j, k;

  // sort by frags
  scoreboardlines = 0;
  for (i = 0; i < CL_MaxClients(); i++) {
    if (cl.scores[i].name[0]) {
      fragsort[scoreboardlines] = i;
      scoreboardlines++;
    }
  }

  for (i = 0; i < scoreboardlines; i++) {
    for (j = 0; j < scoreboardlines - 1 - i; j++) {
      if (CL_ScoresFrags(fragsort[j]) < CL_ScoresFrags(fragsort[j + 1])) {
        k = fragsort[j];
        fragsort[j] = fragsort[j + 1];
        fragsort[j + 1] = k;
      }
    }
  }
}

int Sbar_ColorForMap(int m) { return m + 8; }

//=============================================================================

/*
THERJAK: STILL MISSING
void Sbar_DrawFace(void) {
  // PGM 01/19/97 - team color drawing
  // PGM 03/02/97 - fixed so color swatch only appears in CTF modes
  if (CMLRogue() && (CL_MaxClients() != 1) && (Cvar_GetValue(&teamplay) > 3) &&
      (Cvar_GetValue(&teamplay) < 7)) {
    int top, bottom;
    int xofs;
    char num[12];
    k = CL_Viewentity() -1;
    // draw background
    top = CL_ScoresColors(k) & 0xf0;
    bottom = (CL_ScoresColors(k) & 15) << 4;
    top = Sbar_ColorForMap(top);
    bottom = Sbar_ColorForMap(bottom);

    if (CL_GameTypeDeathMatch())
      xofs = 113;
    else
      xofs = ((ScreenWidth() - 320) >> 1) + 113;

    Draw_Pic(112, 24, rsb_teambord);
    Draw_Fill(xofs, 24 + 3, 22, 9, top, 1);
    Draw_Fill(xofs, 24 + 12, 22, 9, bottom, 1);

    // draw number
    f = CL_ScoresFrags(k);
    sprintf(num, "%3i", f);

    if (top == 8) {
      if (num[0] != ' ') Draw_Character(113, 3 + 24, 18 + num[0] - '0');
      if (num[1] != ' ') Draw_Character(120, 3 + 24, 18 + num[1] - '0');
      if (num[2] != ' ') Draw_Character(127, 3 + 24, 18 + num[2] - '0');
    } else {
      Draw_Character(113, 3 + 24, num[0]);
      Draw_Character(120, 3 + 24, num[1]);
      Draw_Character(127, 3 + 24, num[2]);
    }

    return;
  }
}
*/
/*
===============
Sbar_Draw
===============
*/
void Sbar_Draw(void) {
  float w;  // johnfitz

  if (GetScreenConsoleCurrentHeight() == ScreenHeight())
    return;  // console is full screen

  if (CL_Intermission())
    return;  // johnfitz -- never draw sbar during intermission

  if (SBUpdates() >= GetNumPages() && !Cvar_GetValue(&gl_clear) &&
      Cvar_GetValue(&scr_sbaralpha) >= 1  // johnfitz -- gl_clear, scr_sbaralpha
      && !(Cvar_GetValue(&vid_gamma) != 1)) {
    // ericw -- must draw sbar every frame if doing glsl gamma
    return;
  }

  SBUpdatesInc();

  GL_SetCanvas(CANVAS_DEFAULT);  // johnfitz

  // johnfitz -- don't waste fillrate by clearing the area behind the sbar
  w = CLAMP(320.0f, Cvar_GetValue(&scr_sbarscale) * 320.0f, (float)GL_Width());
  if (Sbar_Lines() && GL_Width() > w) {
    if (Cvar_GetValue(&scr_sbaralpha) < 1)
      Draw_TileClear(0, GL_Height() - Sbar_Lines(), GL_Width(), Sbar_Lines());
    if (CL_GameTypeDeathMatch())
      Draw_TileClear(w, GL_Height() - Sbar_Lines(), GL_Width() - w, Sbar_Lines());
    else {
      Draw_TileClear(0, GL_Height() - Sbar_Lines(), (GL_Width() - w) / 2.0f,
                     Sbar_Lines());
      Draw_TileClear((GL_Width() - w) / 2.0f + w, GL_Height() - Sbar_Lines(),
                     (GL_Width() - w) / 2.0f, Sbar_Lines());
    }
  }
  // johnfitz

  GL_SetCanvas(CANVAS_SBAR);  // johnfitz

  if (Cvar_GetValue(&scr_viewsize) < 110)  {
    Sbar_DrawInventory();
    if (CL_MaxClients() != 1) {
      Sbar_DrawFrags();
    }
  }

  if (Sbar_DoesShowScores() || CL_Stats(STAT_HEALTH) <= 0) {
    Draw_PicAlpha(0, 24, sb_scorebar,
                  Cvar_GetValue(&scr_sbaralpha));
    Sbar_DrawScoreboard();
    SBResetUpdates();
  } else if (Cvar_GetValue(&scr_viewsize) < 120) {
    Draw_PicAlpha(0, 24, sb_sbar,
                  Cvar_GetValue(&scr_sbaralpha));  // johnfitz -- scr_sbaralpha

    // keys (hipnotic only)
    // MED 01/04/97 moved keys here so they would not be overwritten
    if (CMLHipnotic()) {
      if (CL_HasItem(IT_KEY1)) Draw_Pic(209, 3 + 24, sb_items[0]);
      if (CL_HasItem(IT_KEY2)) Draw_Pic(209, 12 + 24, sb_items[1]);
    }
    // armor
    Sbar_DrawArmor();

    // face
    Sbar_DrawFace();

    // health
    Sbar_DrawHealth();

    // ammo icon
    Sbar_DrawAmmo();
  }

  if (CL_GameTypeDeathMatch()) Sbar_MiniDeathmatchOverlay();
}


