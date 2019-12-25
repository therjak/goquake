#include "arch_def.h"
#include "quakedef.h"

/* key up events are sent even if in console mode */

#define HISTORY_FILE_NAME "history.txt"
#define CMDLINES 32

char key_lines[CMDLINES][MAXCMDLINE];  // therjak: extern

int key_linepos;       // therjak: extern
int key_insert;        // therjak: extern
double key_blinktime;  // therjak: extern

int edit_line = 0;     // therjak: extern
int history_line = 0;  // therjak: extern

char *keybindings[MAX_KEYS];
qboolean keydown[MAX_KEYS];

qboolean Key_ShiftDown() { return keydown[K_SHIFT]; }

/*
====================
Key_Console -- johnfitz -- heavy revision

Interactive line editing and console scrollback
====================
*/
extern char key_tabpartial[MAXCMDLINE];

//============================================================================

void History_Init(void) {
  int i, c;
  FILE *hf;

  for (i = 0; i < CMDLINES; i++) {
    key_lines[i][0] = ']';
    key_lines[i][1] = 0;
  }
  key_linepos = 1;

  hf = fopen(va("%s/%s", Com_Basedir(), HISTORY_FILE_NAME), "rt");
  if (hf != NULL) {
    do {
      i = 1;
      do {
        c = fgetc(hf);
        key_lines[edit_line][i++] = c;
      } while (c != '\r' && c != '\n' && c != EOF && i < MAXCMDLINE);
      key_lines[edit_line][i - 1] = 0;
      edit_line = (edit_line + 1) & (CMDLINES - 1);
      /* for people using a windows-generated history file on unix: */
      if (c == '\r' || c == '\n') {
        do
          c = fgetc(hf);
        while (c == '\r' || c == '\n');
        if (c != EOF)
          ungetc(c, hf);
        else
          c = 0; /* loop once more, otherwise last line is lost */
      }
    } while (c != EOF && edit_line < CMDLINES);
    fclose(hf);

    history_line = edit_line = (edit_line - 1) & (CMDLINES - 1);
    key_lines[edit_line][0] = ']';
    key_lines[edit_line][1] = 0;
  } else {
    Con_Printf("BaseDir: %s", Com_Basedir());
  }
}

void History_Shutdown(void) {
  int i;
  FILE *hf;

  hf = fopen(va("%s/%s", Com_Basedir(), HISTORY_FILE_NAME), "wt");
  if (hf != NULL) {
    i = edit_line;
    do {
      i = (i + 1) & (CMDLINES - 1);
    } while (i != edit_line && !key_lines[i][1]);

    while (i != edit_line && key_lines[i][1]) {
      fprintf(hf, "%s\n", key_lines[i] + 1);
      i = (i + 1) & (CMDLINES - 1);
    }
    fclose(hf);
  }
}

/*
===================
Key_Init
===================
*/
void Key_Init(void) {
  History_Init();

  key_blinktime = HostRealTime();  // johnfitz
}
