// sbar.c -- status bar code

#include "_cgo_export.h"
//
#include "quakedef.h"

#define STAT_MINUS 10  // num frame for '-' stats digit

qpic_t *sb_nums[2][11];
qpic_t *sb_sbar;
qpic_t *sb_scorebar;

qpic_t *sb_armor[3];
qpic_t *sb_items[32];

int sb_lines;  // scan lines to draw

qpic_t *rsb_teambord;  // PGM 01/19/97 - team color border

int hipweapons[4] = {HIT_LASER_CANNON_BIT, HIT_MJOLNIR_BIT, 4,
                     HIT_PROXIMITY_GUN_BIT};
// MED 01/04/97 added hipnotic items array

void Sbar_MiniDeathmatchOverlay(void);
void Sbar_DeathmatchOverlay(void);

/*
===============
Sbar_LoadPics -- johnfitz -- load all the sbar pics
===============
*/
void Sbar_LoadPics(void) {
  int i;

  for (i = 0; i < 10; i++) {
    sb_nums[0][i] = Draw_PicFromWad(va("num_%i", i));
    sb_nums[1][i] = Draw_PicFromWad(va("anum_%i", i));
  }

  sb_nums[0][10] = Draw_PicFromWad("num_minus");
  sb_nums[1][10] = Draw_PicFromWad("anum_minus");

  sb_armor[0] = Draw_PicFromWad("sb_armor1");
  sb_armor[1] = Draw_PicFromWad("sb_armor2");
  sb_armor[2] = Draw_PicFromWad("sb_armor3");

  sb_items[0] = Draw_PicFromWad("sb_key1");
  sb_items[1] = Draw_PicFromWad("sb_key2");

  sb_sbar = Draw_PicFromWad("sbar");
  sb_scorebar = Draw_PicFromWad("scorebar");

  if (CMLRogue()) {
    rsb_teambord = Draw_PicFromWad("r_teambord");

  }
}

/*
=============
Sbar_itoa
=============
*/
int Sbar_itoa(int num, char *buf) {
  char *str;
  int pow10;
  int dig;

  str = buf;

  if (num < 0) {
    *str++ = '-';
    num = -num;
  }

  for (pow10 = 10; num >= pow10; pow10 *= 10)
    ;

  do {
    pow10 /= 10;
    dig = num / pow10;
    *str++ = '0' + dig;
    num -= dig * pow10;
  } while (pow10 != 1);

  *str = 0;

  return str - buf;
}

/*
=============
Sbar_DrawNum
=============
*/
void Sbar_DrawNum(int x, int y, int num, int digits, int color) {
  char str[12];
  char *ptr;
  int l, frame;

  num = q_min(999, num);

  l = Sbar_itoa(num, str);
  ptr = str;
  if (l > digits) ptr += (l - digits);
  if (l < digits) x += (digits - l) * 24;

  while (*ptr) {
    if (*ptr == '-')
      frame = STAT_MINUS;
    else
      frame = *ptr - '0';

    Draw_Pic(x, y + 24, sb_nums[color][frame]);
    x += 24;
    ptr++;
  }
}

//=============================================================================

int fragsort[MAX_SCOREBOARD];

char scoreboardtext[MAX_SCOREBOARD][20];
int scoreboardtop[MAX_SCOREBOARD];
int scoreboardbottom[MAX_SCOREBOARD];
int scoreboardcount[MAX_SCOREBOARD];
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

/*
===============
Sbar_UpdateScoreboard
===============
*/
void Sbar_UpdateScoreboard(void) {
  int i, k;
  int top, bottom;
  scoreboard_t *s;

  Sbar_SortFrags();

  // draw the text
  memset(scoreboardtext, 0, sizeof(scoreboardtext));

  for (i = 0; i < scoreboardlines; i++) {
    k = fragsort[i];
    s = &cl.scores[k];
    sprintf(&scoreboardtext[i][1], "%3i %s", CL_ScoresFrags(k), s->name);

    top = CL_ScoresColors(k) & 0xf0;
    bottom = (CL_ScoresColors(k) & 15) << 4;
    scoreboardtop[i] = Sbar_ColorForMap(top);
    scoreboardbottom[i] = Sbar_ColorForMap(bottom);
  }
}

/*
===============
Sbar_DrawScoreboard
===============
*/
void Sbar_DrawScoreboard(void) {
  Sbar_SoloScoreboard();
  if (CL_GameTypeDeathMatch()) Sbar_DeathmatchOverlay();
}

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
  if (sb_lines && GL_Width() > w) {
    if (Cvar_GetValue(&scr_sbaralpha) < 1)
      Draw_TileClear(0, GL_Height() - sb_lines, GL_Width(), sb_lines);
    if (CL_GameTypeDeathMatch())
      Draw_TileClear(w, GL_Height() - sb_lines, GL_Width() - w, sb_lines);
    else {
      Draw_TileClear(0, GL_Height() - sb_lines, (GL_Width() - w) / 2.0f,
                     sb_lines);
      Draw_TileClear((GL_Width() - w) / 2.0f + w, GL_Height() - sb_lines,
                     (GL_Width() - w) / 2.0f, sb_lines);
    }
  }
  // johnfitz

  GL_SetCanvas(CANVAS_SBAR);  // johnfitz

  if (Cvar_GetValue(&scr_viewsize) <
      110)  // johnfitz -- check viewsize instead of sb_lines
  {
    Sbar_DrawInventory();
    if (CL_MaxClients() != 1) Sbar_DrawFrags();
  }

  if (Sbar_DoesShowScores() || CL_Stats(STAT_HEALTH) <= 0) {
    Draw_PicAlpha(0, 24, sb_scorebar,
                  Cvar_GetValue(&scr_sbaralpha));  // johnfitz -- scr_sbaralpha
    Sbar_DrawScoreboard();
    SBResetUpdates();
  } else if (Cvar_GetValue(&scr_viewsize) <
             120)  // johnfitz -- check viewsize instead of sb_lines
  {
    Draw_PicAlpha(0, 24, sb_sbar,
                  Cvar_GetValue(&scr_sbaralpha));  // johnfitz -- scr_sbaralpha

    // keys (hipnotic only)
    // MED 01/04/97 moved keys here so they would not be overwritten
    if (CMLHipnotic()) {
      if (CL_HasItem(IT_KEY1)) Draw_Pic(209, 3 + 24, sb_items[0]);
      if (CL_HasItem(IT_KEY2)) Draw_Pic(209, 12 + 24, sb_items[1]);
    }
    // armor
    if (CL_HasItem(IT_INVULNERABILITY)) {
      Sbar_DrawNum(24, 0, 666, 3, 1);
      Draw_Pic(0, 24, draw_disc);
    } else {
      if (CMLRogue()) {
        Sbar_DrawNum(24, 0, CL_Stats(STAT_ARMOR), 3,
                     CL_Stats(STAT_ARMOR) <= 25);
        if (CL_HasItem(RIT_ARMOR3))
          Draw_Pic(0, 24, sb_armor[2]);
        else if (CL_HasItem(RIT_ARMOR2))
          Draw_Pic(0, 24, sb_armor[1]);
        else if (CL_HasItem(RIT_ARMOR1))
          Draw_Pic(0, 24, sb_armor[0]);
      } else {
        Sbar_DrawNum(24, 0, CL_Stats(STAT_ARMOR), 3,
                     CL_Stats(STAT_ARMOR) <= 25);
        if (CL_HasItem(IT_ARMOR3))
          Draw_Pic(0, 24, sb_armor[2]);
        else if (CL_HasItem(IT_ARMOR2))
          Draw_Pic(0, 24, sb_armor[1]);
        else if (CL_HasItem(IT_ARMOR1))
          Draw_Pic(0, 24, sb_armor[0]);
      }
    }

    // face
    Sbar_DrawFace();

    // health
    Sbar_DrawHealth();

    // ammo icon
    Sbar_DrawAmmo();
  }

  if (CL_GameTypeDeathMatch()) Sbar_MiniDeathmatchOverlay();
}

/*
==================
Sbar_MiniDeathmatchOverlay
==================
*/
void Sbar_MiniDeathmatchOverlay(void) {
  int i, k, top, bottom, x, y, f, numlines;
  char num[12];
  float scale;  // johnfitz
  scoreboard_t *s;

  scale = CLAMP(1.0, Cvar_GetValue(&scr_sbarscale),
                (float)GL_Width() / 320.0);  // johnfitz

  // MAX_SCOREBOARDNAME = 32, so total width for this overlay plus sbar is 632,
  // but we can cut off some i guess
  if (GL_Width() / scale < 512 ||
      Cvar_GetValue(&scr_viewsize) >=
          120)  // johnfitz -- test should consider scr_sbarscale
    return;

  // scores
  Sbar_SortFrags();

  // draw the text
  numlines = (Cvar_GetValue(&scr_viewsize) >= 110) ? 3 : 6;  // johnfitz

  // find us
  for (i = 0; i < scoreboardlines; i++)
    if (fragsort[i] == CL_Viewentity() - 1) break;
  if (i == scoreboardlines)  // we're not there
    i = 0;
  else  // figure out start
    i = i - numlines / 2;
  if (i > scoreboardlines - numlines) i = scoreboardlines - numlines;
  if (i < 0) i = 0;

  x = 324;
  y = (Cvar_GetValue(&scr_viewsize) >= 110)
          ? 24
          : 0;  // johnfitz -- start at the right place
  for (; i < scoreboardlines && y <= 48;
       i++, y += 8)  // johnfitz -- change y init, test, inc
  {
    k = fragsort[i];
    s = &cl.scores[k];
    if (!s->name[0]) continue;

    // colors
    top = CL_ScoresColors(k) & 0xf0;
    bottom = (CL_ScoresColors(k) & 15) << 4;
    top = Sbar_ColorForMap(top);
    bottom = Sbar_ColorForMap(bottom);

    Draw_Fill(x, y + 1, 40, 4, top, 1);
    Draw_Fill(x, y + 5, 40, 3, bottom, 1);

    // number
    f = CL_ScoresFrags(k);
    sprintf(num, "%3i", f);
    Draw_Character(x + 8, y, num[0]);
    Draw_Character(x + 16, y, num[1]);
    Draw_Character(x + 24, y, num[2]);

    // brackets
    if (k == CL_Viewentity() - 1) {
      Draw_Character(x, y, 16);
      Draw_Character(x + 32, y, 17);
    }

    // name
    Draw_String(x + 48, y, s->name);
  }
}


