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

typedef struct {
  const char *name;
  int keynum;
} keyname_t;

keyname_t keynames[] = {
    {"TAB", K_TAB},
    {"ENTER", K_ENTER},
    {"ESCAPE", K_ESCAPE},
    {"SPACE", K_SPACE},
    {"BACKSPACE", K_BACKSPACE},
    {"UPARROW", K_UPARROW},
    {"DOWNARROW", K_DOWNARROW},
    {"LEFTARROW", K_LEFTARROW},
    {"RIGHTARROW", K_RIGHTARROW},

    {"ALT", K_ALT},
    {"CTRL", K_CTRL},
    {"SHIFT", K_SHIFT},

    //	{"KP_NUMLOCK", K_KP_NUMLOCK},
    {"KP_SLASH", K_KP_SLASH},
    {"KP_STAR", K_KP_STAR},
    {"KP_MINUS", K_KP_MINUS},
    {"KP_HOME", K_KP_HOME},
    {"KP_UPARROW", K_KP_UPARROW},
    {"KP_PGUP", K_KP_PGUP},
    {"KP_PLUS", K_KP_PLUS},
    {"KP_LEFTARROW", K_KP_LEFTARROW},
    {"KP_5", K_KP_5},
    {"KP_RIGHTARROW", K_KP_RIGHTARROW},
    {"KP_END", K_KP_END},
    {"KP_DOWNARROW", K_KP_DOWNARROW},
    {"KP_PGDN", K_KP_PGDN},
    {"KP_ENTER", K_KP_ENTER},
    {"KP_INS", K_KP_INS},
    {"KP_DEL", K_KP_DEL},

    {"F1", K_F1},
    {"F2", K_F2},
    {"F3", K_F3},
    {"F4", K_F4},
    {"F5", K_F5},
    {"F6", K_F6},
    {"F7", K_F7},
    {"F8", K_F8},
    {"F9", K_F9},
    {"F10", K_F10},
    {"F11", K_F11},
    {"F12", K_F12},

    {"INS", K_INS},
    {"DEL", K_DEL},
    {"PGDN", K_PGDN},
    {"PGUP", K_PGUP},
    {"HOME", K_HOME},
    {"END", K_END},

    {"COMMAND", K_COMMAND},

    {"MOUSE1", K_MOUSE1},
    {"MOUSE2", K_MOUSE2},
    {"MOUSE3", K_MOUSE3},
    {"MOUSE4", K_MOUSE4},
    {"MOUSE5", K_MOUSE5},

    {"JOY1", K_JOY1},
    {"JOY2", K_JOY2},
    {"JOY3", K_JOY3},
    {"JOY4", K_JOY4},

    {"AUX1", K_AUX1},
    {"AUX2", K_AUX2},
    {"AUX3", K_AUX3},
    {"AUX4", K_AUX4},
    {"AUX5", K_AUX5},
    {"AUX6", K_AUX6},
    {"AUX7", K_AUX7},
    {"AUX8", K_AUX8},
    {"AUX9", K_AUX9},
    {"AUX10", K_AUX10},
    {"AUX11", K_AUX11},
    {"AUX12", K_AUX12},
    {"AUX13", K_AUX13},
    {"AUX14", K_AUX14},
    {"AUX15", K_AUX15},
    {"AUX16", K_AUX16},
    {"AUX17", K_AUX17},
    {"AUX18", K_AUX18},
    {"AUX19", K_AUX19},
    {"AUX20", K_AUX20},
    {"AUX21", K_AUX21},
    {"AUX22", K_AUX22},
    {"AUX23", K_AUX23},
    {"AUX24", K_AUX24},
    {"AUX25", K_AUX25},
    {"AUX26", K_AUX26},
    {"AUX27", K_AUX27},
    {"AUX28", K_AUX28},
    {"AUX29", K_AUX29},
    {"AUX30", K_AUX30},
    {"AUX31", K_AUX31},
    {"AUX32", K_AUX32},

    {"PAUSE", K_PAUSE},

    {"MWHEELUP", K_MWHEELUP},
    {"MWHEELDOWN", K_MWHEELDOWN},

    {"SEMICOLON", ';'},  // because a raw semicolon seperates commands

    {"BACKQUOTE", '`'},  // because a raw backquote may toggle the console
    {"TILDE", '~'},      // because a raw tilde may toggle the console

    {"LTHUMB", K_LTHUMB},
    {"RTHUMB", K_RTHUMB},
    {"LSHOULDER", K_LSHOULDER},
    {"RSHOULDER", K_RSHOULDER},
    {"ABUTTON", K_ABUTTON},
    {"BBUTTON", K_BBUTTON},
    {"XBUTTON", K_XBUTTON},
    {"YBUTTON", K_YBUTTON},
    {"LTRIGGER", K_LTRIGGER},
    {"RTRIGGER", K_RTRIGGER},

    {NULL, 0}};

/*
==============================================================================

                        LINE TYPING INTO THE CONSOLE

==============================================================================
*/

static void PasteToConsole(void) {
  char *cbd, *p, *workline;
  int mvlen, inslen;

  if (key_linepos == MAXCMDLINE - 1) return;

  if ((cbd = PL_GetClipboardData()) == NULL) return;

  p = cbd;
  while (*p) {
    if (*p == '\n' || *p == '\r' || *p == '\b') {
      *p = 0;
      break;
    }
    p++;
  }

  inslen = (int)(p - cbd);
  if (inslen + key_linepos > MAXCMDLINE - 1)
    inslen = MAXCMDLINE - 1 - key_linepos;
  if (inslen <= 0) goto done;

  workline = key_lines[edit_line];
  workline += key_linepos;
  mvlen = (int)strlen(workline);
  if (mvlen + inslen + key_linepos > MAXCMDLINE - 1) {
    mvlen = MAXCMDLINE - 1 - key_linepos - inslen;
    if (mvlen < 0) mvlen = 0;
  }

  // insert the string
  if (mvlen != 0) memmove(workline + inslen, workline, mvlen);
  memcpy(workline, cbd, inslen);
  key_linepos += inslen;
  workline[mvlen + inslen] = '\0';
done:
  free(cbd);
}

/*
====================
Key_Console -- johnfitz -- heavy revision

Interactive line editing and console scrollback
====================
*/
extern char key_tabpartial[MAXCMDLINE];

//============================================================================

qboolean chat_team = false;  // therjak: extern
static char chat_buffer[MAXCMDLINE];
static int chat_bufferlen = 0;

const char *Key_GetChatBuffer(void) { return chat_buffer; }

int Key_GetChatMsgLen(void) { return chat_bufferlen; }

void Key_EndChat(void) {
  SetKeyDest(key_game);
  chat_bufferlen = 0;
  chat_buffer[0] = 0;
}

void Key_Message(int key) {
  switch (key) {
    case K_ENTER:
    case K_KP_ENTER:
      if (chat_team)
        Cbuf_AddText("say_team \"");
      else
        Cbuf_AddText("say \"");
      Cbuf_AddText(chat_buffer);
      Cbuf_AddText("\"\n");

      Key_EndChat();
      return;

    case K_ESCAPE:
      Key_EndChat();
      return;

    case K_BACKSPACE:
      if (chat_bufferlen) chat_buffer[--chat_bufferlen] = 0;
      return;
  }
}

void Char_Message(int key) {
  if (chat_bufferlen == sizeof(chat_buffer) - 1) return;  // all full

  chat_buffer[chat_bufferlen++] = key;
  chat_buffer[chat_bufferlen] = 0;
}

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
