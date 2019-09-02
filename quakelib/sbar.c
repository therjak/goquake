// sbar.c -- status bar code

#include "_cgo_export.h"
//
#include "quakedef.h"

#define STAT_MINUS 10  // num frame for '-' stats digit

qpic_t *sb_nums[2][11];
qpic_t *sb_colon, *sb_slash;
qpic_t *sb_sbar;
qpic_t *sb_scorebar;

qpic_t *sb_ammo[4];
qpic_t *sb_sigil[4];
qpic_t *sb_armor[3];
qpic_t *sb_items[32];

qpic_t *sb_faces[7][2];  // 0 is gibbed, 1 is dead, 2-6 are alive
                         // 0 is static, 1 is temporary animation
qpic_t *sb_face_invis;
qpic_t *sb_face_quad;
qpic_t *sb_face_invuln;
qpic_t *sb_face_invis_invuln;

int sb_lines;  // scan lines to draw

qpic_t *rsb_ammo[3];
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

  sb_colon = Draw_PicFromWad("num_colon");
  sb_slash = Draw_PicFromWad("num_slash");

  sb_ammo[0] = Draw_PicFromWad("sb_shells");
  sb_ammo[1] = Draw_PicFromWad("sb_nails");
  sb_ammo[2] = Draw_PicFromWad("sb_rocket");
  sb_ammo[3] = Draw_PicFromWad("sb_cells");

  sb_armor[0] = Draw_PicFromWad("sb_armor1");
  sb_armor[1] = Draw_PicFromWad("sb_armor2");
  sb_armor[2] = Draw_PicFromWad("sb_armor3");

  sb_items[0] = Draw_PicFromWad("sb_key1");
  sb_items[1] = Draw_PicFromWad("sb_key2");
  sb_items[2] = Draw_PicFromWad("sb_invis");
  sb_items[3] = Draw_PicFromWad("sb_invuln");
  sb_items[4] = Draw_PicFromWad("sb_suit");
  sb_items[5] = Draw_PicFromWad("sb_quad");

  sb_sigil[0] = Draw_PicFromWad("sb_sigil1");
  sb_sigil[1] = Draw_PicFromWad("sb_sigil2");
  sb_sigil[2] = Draw_PicFromWad("sb_sigil3");
  sb_sigil[3] = Draw_PicFromWad("sb_sigil4");

  sb_faces[4][0] = Draw_PicFromWad("face1");
  sb_faces[4][1] = Draw_PicFromWad("face_p1");
  sb_faces[3][0] = Draw_PicFromWad("face2");
  sb_faces[3][1] = Draw_PicFromWad("face_p2");
  sb_faces[2][0] = Draw_PicFromWad("face3");
  sb_faces[2][1] = Draw_PicFromWad("face_p3");
  sb_faces[1][0] = Draw_PicFromWad("face4");
  sb_faces[1][1] = Draw_PicFromWad("face_p4");
  sb_faces[0][0] = Draw_PicFromWad("face5");
  sb_faces[0][1] = Draw_PicFromWad("face_p5");

  sb_face_invis = Draw_PicFromWad("face_invis");
  sb_face_invuln = Draw_PicFromWad("face_invul2");
  sb_face_invis_invuln = Draw_PicFromWad("face_inv2");
  sb_face_quad = Draw_PicFromWad("face_quad");

  sb_sbar = Draw_PicFromWad("sbar");
  sb_scorebar = Draw_PicFromWad("scorebar");

  if (CMLRogue()) {
    rsb_teambord = Draw_PicFromWad("r_teambord");

    rsb_ammo[0] = Draw_PicFromWad("r_ammolava");
    rsb_ammo[1] = Draw_PicFromWad("r_ammomulti");
    rsb_ammo[2] = Draw_PicFromWad("r_ammoplasma");
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
Sbar_SoloScoreboard -- johnfitz -- new layout
===============
*/
void Sbar_SoloScoreboard(void) {
  char str[256];
  int minutes, seconds, tens, units;

  sprintf(str, "Kills: %i/%i", CL_Stats(STAT_MONSTERS),
          CL_Stats(STAT_TOTALMONSTERS));
  Draw_String(8, 12 + 24, str);

  sprintf(str, "Secrets: %i/%i", CL_Stats(STAT_SECRETS),
          CL_Stats(STAT_TOTALSECRETS));
  Draw_String(312 - strlen(str) * 8, 12 + 24, str);

  if (!CMLFitz()) { /* QuakeSpasm customization: */
    q_snprintf(str, sizeof(str), "skill %i",
               (int)(Cvar_GetValue(&skill) + 0.5));
    Draw_String(160 - strlen(str) * 4, 12 + 24, str);

    q_snprintf(str, sizeof(str), "%s (%s)", cl.levelname, cl.mapname);
    Sbar_DrawScrollString(0, 4 + 24, 320, str);
    return;
  }
  minutes = CL_Time() / 60;
  seconds = CL_Time() - 60 * minutes;
  tens = seconds / 10;
  units = seconds - 10 * tens;
  sprintf(str, "%i:%i%i", minutes, tens, units);
  Draw_String(160 - strlen(str) * 4, 12 + 24, str);

  Sbar_DrawScrollString(0, 4 + 24, 320, cl.levelname);
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
    Sbar_DrawNum(136, 0, CL_Stats(STAT_HEALTH), 3, CL_Stats(STAT_HEALTH) <= 25);

    // ammo icon
    if (CMLRogue()) {
      if (CL_HasItem(RIT_SHELLS))
        Draw_Pic(224, 24, sb_ammo[0]);
      else if (CL_HasItem(RIT_NAILS))
        Draw_Pic(224, 24, sb_ammo[1]);
      else if (CL_HasItem(RIT_ROCKETS))
        Draw_Pic(224, 24, sb_ammo[2]);
      else if (CL_HasItem(RIT_CELLS))
        Draw_Pic(224, 24, sb_ammo[3]);
      else if (CL_HasItem(RIT_LAVA_NAILS))
        Draw_Pic(224, 24, rsb_ammo[0]);
      else if (CL_HasItem(RIT_PLASMA_AMMO))
        Draw_Pic(224, 24, rsb_ammo[1]);
      else if (CL_HasItem(RIT_MULTI_ROCKETS))
        Draw_Pic(224, 24, rsb_ammo[2]);
    } else {
      if (CL_HasItem(IT_SHELLS))
        Draw_Pic(224, 24, sb_ammo[0]);
      else if (CL_HasItem(IT_NAILS))
        Draw_Pic(224, 24, sb_ammo[1]);
      else if (CL_HasItem(IT_ROCKETS))
        Draw_Pic(224, 24, sb_ammo[2]);
      else if (CL_HasItem(IT_CELLS))
        Draw_Pic(224, 24, sb_ammo[3]);
    }

    Sbar_DrawNum(248, 0, CL_Stats(STAT_AMMO), 3, CL_Stats(STAT_AMMO) <= 10);
  }

  if (CL_GameTypeDeathMatch()) Sbar_MiniDeathmatchOverlay();
}

//=============================================================================

/*
==================
Sbar_IntermissionNumber

==================
*/
void Sbar_IntermissionNumber(int x, int y, int num, int digits, int color) {
  char str[12];
  char *ptr;
  int l, frame;

  l = Sbar_itoa(num, str);
  ptr = str;
  if (l > digits) ptr += (l - digits);
  if (l < digits) x += (digits - l) * 24;

  while (*ptr) {
    if (*ptr == '-')
      frame = STAT_MINUS;
    else
      frame = *ptr - '0';

    Draw_Pic(x, y, sb_nums[color][frame]);  // johnfitz -- stretched menus
    x += 24;
    ptr++;
  }
}

/*
==================
Sbar_DeathmatchOverlay

==================
*/
void Sbar_DeathmatchOverlay(void) {
  qpic_t *pic;
  int i, k, l;
  int top, bottom;
  int x, y, f;
  char num[12];
  scoreboard_t *s;

  GL_SetCanvas(CANVAS_MENU);

  pic = Draw_CachePic("gfx/ranking.lmp");
  Draw_Pic((320 - pic->width) / 2, 8, pic);

  // scores
  Sbar_SortFrags();

  // draw the text
  l = scoreboardlines;

  x = 80;
  y = 40;
  for (i = 0; i < l; i++) {
    k = fragsort[i];
    s = &cl.scores[k];
    if (!s->name[0]) continue;

    // draw background
    top = CL_ScoresColors(k) & 0xf0;
    bottom = (CL_ScoresColors(k) & 15) << 4;
    top = Sbar_ColorForMap(top);
    bottom = Sbar_ColorForMap(bottom);

    Draw_Fill(x, y, 40, 4, top, 1);
    Draw_Fill(x, y + 4, 40, 4, bottom, 1);

    // draw number
    f = CL_ScoresFrags(k);
    sprintf(num, "%3i", f);

    Draw_Character(x + 8, y, num[0]);
    Draw_Character(x + 16, y, num[1]);
    Draw_Character(x + 24, y, num[2]);

    if (k == CL_Viewentity() - 1) Draw_Character(x - 8, y, 12);

    // draw name
    M_Print(x + 64, y, s->name);

    y += 10;
  }

  GL_SetCanvas(CANVAS_SBAR);
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

/*
==================
Sbar_IntermissionOverlay
==================
*/
void Sbar_IntermissionOverlay(void) {
  qpic_t *pic;
  int dig;
  int num;

  if (CL_GameTypeDeathMatch()) {
    Sbar_DeathmatchOverlay();
    return;
  }

  GL_SetCanvas(CANVAS_MENU);

  pic = Draw_CachePic("gfx/complete.lmp");
  Draw_Pic(64, 24, pic);

  pic = Draw_CachePic("gfx/inter.lmp");
  Draw_Pic(0, 56, pic);

  dig = CL_CompletedTime() / 60;
  Sbar_IntermissionNumber(152, 64, dig, 3, 0);
  num = CL_CompletedTime() - dig * 60;
  Draw_Pic(224, 64, sb_colon);
  Draw_Pic(240, 64, sb_nums[0][num / 10]);
  Draw_Pic(264, 64, sb_nums[0][num % 10]);

  Sbar_IntermissionNumber(152, 104, CL_Stats(STAT_SECRETS), 3, 0);
  Draw_Pic(224, 104, sb_slash);
  Sbar_IntermissionNumber(240, 104, CL_Stats(STAT_TOTALSECRETS), 3, 0);

  Sbar_IntermissionNumber(152, 144, CL_Stats(STAT_MONSTERS), 3, 0);
  Draw_Pic(224, 144, sb_slash);
  Sbar_IntermissionNumber(240, 144, CL_Stats(STAT_TOTALMONSTERS), 3, 0);
}

/*
==================
Sbar_FinaleOverlay
==================
*/
void Sbar_FinaleOverlay(void) {
  qpic_t *pic;

  GL_SetCanvas(CANVAS_MENU);

  pic = Draw_CachePic("gfx/finale.lmp");
  Draw_Pic((320 - pic->width) / 2, 16, pic);
}
