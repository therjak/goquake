#include "arch_def.h"
#include "quakedef.h"

/* key up events are sent even if in console mode */

#define HISTORY_FILE_NAME "history.txt"
#define CMDLINES 32

char key_lines[CMDLINES][MAXCMDLINE];  // therjak: extern

int key_linepos;       // therjak: extern
double key_blinktime;  // therjak: extern

int edit_line = 0;     // therjak: extern
int history_line = 0;  // therjak: extern

qboolean keydown[MAX_KEYS];

qboolean Key_ShiftDown() { return keydown[K_SHIFT]; }

void Key_Init(void) {
  History_Init();

  key_blinktime = HostRealTime();  // johnfitz
}
