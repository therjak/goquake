#include <fcntl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <time.h>
#ifdef _WIN32
#include <io.h>
#else
#include <unistd.h>
#endif
#include "quakedef.h"

// Needs:
// COM_CreatePath
// Key_GetChatMsgLen
// Key_GetChatBuffer
// SCR_EndLoadingPlaque
// SCR_UpdateScreen
//
// Has:
// GL_SetCanvas
// Draw_String
// Draw_Pic
// Draw_Character
// CLS_GetState --
// CLS_IsDemoPlayback --
// CLS_GetSignon --
// Cmd_AddCommand --
// Con_SafePrintf --
// Cvar_GetValue --
// IN_Activate --
// IN_Deactivate --
// S_LocalSound --
// ScreenDisabled --
// SetScreenDisabled --
// Sys_Print --

float con_cursorspeed = 4;

#define CON_TEXTSIZE 65536  // johnfitz -- new default size
#define CON_MINSIZE 16384   // johnfitz -- old default, now the minimum size

int con_buffersize;  // johnfitz -- user can now override default

int con_totallines;  // total lines in console scrollback
int con_backscroll;  // lines up from bottom to display
int con_current;     // where next message will be printed
int con_x;           // offset in current line for next print
char *con_text = NULL;

cvar_t con_notifytime;  // = {"con_notifytime", "3", CVAR_NONE};          //
                        // seconds

#define NUM_CON_TIMES 4
float con_times[NUM_CON_TIMES];  // realtime time the line was generated
                                 // for transparent notify lines

int con_vislines;

qboolean con_debuglog = false;

/*
================
Con_ToggleConsole_f
================
*/
extern int history_line;  // johnfitz

/*
================
Con_Dump_f -- johnfitz -- adapted from quake2 source
================
*/
static void Con_Dump_f(void) {
  int l, x;
  const char *line;
  FILE *f;
  char buffer[1024];
  char name[MAX_OSPATH];

  q_snprintf(name, sizeof(name), "%s/condump.txt", Com_Gamedir());
  COM_CreatePath(name);
  f = fopen(name, "w");
  if (!f) {
    Con_Printf("ERROR: couldn't open file %s.\n", name);
    return;
  }

  // skip initial empty lines
  for (l = con_current - con_totallines + 1; l <= con_current; l++) {
    line = con_text + (l % con_totallines) * ConsoleWidth();
    for (x = 0; x < ConsoleWidth(); x++)
      if (line[x] != ' ') break;
    if (x != ConsoleWidth()) break;
  }

  // write the remaining lines
  buffer[ConsoleWidth()] = 0;
  for (; l <= con_current; l++) {
    line = con_text + (l % con_totallines) * ConsoleWidth();
    strncpy(buffer, line, ConsoleWidth());
    for (x = ConsoleWidth() - 1; x >= 0; x--) {
      if (buffer[x] == ' ')
        buffer[x] = 0;
      else
        break;
    }
    for (x = 0; buffer[x]; x++) buffer[x] &= 0x7f;

    fprintf(f, "%s\n", buffer);
  }

  fclose(f);
  Con_Printf("Dumped console text to %s.\n", name);
}

/*
================
Con_CheckResize

If the line width has changed, reformat the buffer.
================
*/
void ConCheckResize(void) {
  int i, j, width, oldwidth, oldtotallines, numlines, numchars;
  char *tbuf;  // johnfitz -- tbuf no longer a static array
  int mark;    // johnfitz

  width = (ConWidth() >> 3) - 2;

  if (width == ConsoleWidth()) return;

  oldwidth = ConsoleWidth();
  SetConsoleWidth(width);
  oldtotallines = con_totallines;
  con_totallines = con_buffersize / ConsoleWidth();
  numlines = oldtotallines;

  if (con_totallines < numlines) numlines = con_totallines;

  numchars = oldwidth;

  if (ConsoleWidth() < numchars) numchars = ConsoleWidth();

  mark = Hunk_LowMark();
  tbuf = (char *)Hunk_Alloc(con_buffersize);

  Q_memcpy(tbuf, con_text, con_buffersize);
  Q_memset(con_text, ' ', con_buffersize);

  for (i = 0; i < numlines; i++) {
    for (j = 0; j < numchars; j++) {
      con_text[(con_totallines - 1 - i) * ConsoleWidth() + j] =
          tbuf[((con_current - i + oldtotallines) % oldtotallines) * oldwidth +
               j];
    }
  }

  Hunk_FreeToLowMark(mark);

  Con_ClearNotify();

  con_backscroll = 0;
  con_current = con_totallines - 1;
}

/*
================
Con_Init
================
*/
void ConInit(void) {
  con_buffersize = q_max(CON_MINSIZE, CMLConsoleSize() * 1024);

  con_text = (char *)Hunk_AllocName(
      con_buffersize,
      "context");  // johnfitz -- con_buffersize replaces CON_TEXTSIZE
  Q_memset(con_text, ' ',
           con_buffersize);  // johnfitz -- con_buffersize replaces CON_TEXTSIZE
  // johnfitz -- no need to run Con_CheckResize here
  SetConsoleWidth(38);
  con_totallines = con_buffersize / ConsoleWidth();
  con_backscroll = 0;
  con_current = con_totallines - 1;
  // johnfitz

  Con_Printf("Console initialized.\n");

  Cvar_FakeRegister(&con_notifytime, "con_notifytime");

  Cmd_AddCommand("condump", Con_Dump_f);  // johnfitz
}

/*
===============
Con_Linefeed
===============
*/
static void Con_Linefeed(void) {
  // johnfitz -- improved scrolling
  if (con_backscroll) con_backscroll++;
  if (con_backscroll > con_totallines - (GL_Height() >> 3) - 1)
    con_backscroll = con_totallines - (GL_Height() >> 3) - 1;
  // johnfitz

  con_x = 0;
  con_current++;
  Q_memset(&con_text[(con_current % con_totallines) * ConsoleWidth()], ' ',
           ConsoleWidth());
}

/*
================
Con_Printf

Handles cursor positioning, line wrapping, etc
================
*/
#define MAXPRINTMSG 4096
void Con_Printf(const char *fmt, ...) {
  va_list argptr;
  char msg[MAXPRINTMSG];

  va_start(argptr, fmt);
  q_vsnprintf(msg, sizeof(msg), fmt, argptr);
  va_end(argptr);

  // also echo to debugging console
  Sys_Print(msg);
  Con_PrintStr(msg);
}

/*
================
Con_DWarning -- ericw

same as Con_Warning, but only prints if "developer" cvar is set.
use for "exceeds standard limit of" messages, which are only relevant for
developers
targetting vanilla engines
================
*/
void Con_DWarning(const char *fmt, ...) {
  va_list argptr;
  char msg[MAXPRINTMSG];

  if (!Cvar_GetValue(&developer))
    return;  // don't confuse non-developers with techie stuff...

  va_start(argptr, fmt);
  q_vsnprintf(msg, sizeof(msg), fmt, argptr);
  va_end(argptr);

  Con_SafePrintf("\x02Warning: ");
  Con_Printf("%s", msg);
}

/*
================
Con_Warning -- johnfitz -- prints a warning to the console
================
*/
void Con_Warning(const char *fmt, ...) {
  va_list argptr;
  char msg[MAXPRINTMSG];

  va_start(argptr, fmt);
  q_vsnprintf(msg, sizeof(msg), fmt, argptr);
  va_end(argptr);

  Con_SafePrintf("\x02Warning: ");
  Con_Printf("%s", msg);
}

/*
================
Con_DPrintf

A Con_Printf that only shows up if the "developer" cvar is set
================
*/
void Con_DPrintf(const char *fmt, ...) {
  va_list argptr;
  char msg[MAXPRINTMSG];

  if (!Cvar_GetValue(&developer))
    return;  // don't confuse non-developers with techie stuff...

  va_start(argptr, fmt);
  q_vsnprintf(msg, sizeof(msg), fmt, argptr);
  va_end(argptr);

  Con_SafePrintf("%s", msg);  // johnfitz -- was Con_Printf
}

/*
================
Con_DPrintf2 -- johnfitz -- only prints if "developer" >= 2

currently not used
================
*/
void Con_DPrintf2(const char *fmt, ...) {
  va_list argptr;
  char msg[MAXPRINTMSG];

  if (Cvar_GetValue(&developer) >= 2) {
    va_start(argptr, fmt);
    q_vsnprintf(msg, sizeof(msg), fmt, argptr);
    va_end(argptr);
    Con_Printf("%s", msg);
  }
}

/*
==================
Con_SafePrintf

Okay to call even when the screen can't be updated
==================
*/
void Con_SafePrintf(const char *fmt, ...) {
  va_list argptr;
  char msg[1024];
  int temp;

  va_start(argptr, fmt);
  q_vsnprintf(msg, sizeof(msg), fmt, argptr);
  va_end(argptr);

  temp = ScreenDisabled();
  SetScreenDisabled(true);
  Con_Printf("%s", msg);
  SetScreenDisabled(temp);
}

/*
==============================================================================

        TAB COMPLETION

==============================================================================
*/

// johnfitz -- tab completion stuff
// unique defs
char key_tabpartial[MAXCMDLINE];
typedef struct tab_s {
  const char *name;
  const char *type;
  struct tab_s *next;
  struct tab_s *prev;
} tab_t;
tab_t *tablist;

/*
============
AddToTabList -- johnfitz

tablist is a doubly-linked loop, alphabetized by name
============
*/

// bash_partial is the string that can be expanded,
// aka Linux Bash shell. -- S.A.
static char bash_partial[80];
static qboolean bash_singlematch;

void AddToTabList(const char *name, const char *type) {
  tab_t *t, *insert;
  char *i_bash;
  const char *i_name;

  if (!*bash_partial) {
    strncpy(bash_partial, name, 79);
    bash_partial[79] = '\0';
  } else {
    bash_singlematch = 0;
    // find max common between bash_partial and name
    i_bash = bash_partial;
    i_name = name;
    while (*i_bash && (*i_bash == *i_name)) {
      i_bash++;
      i_name++;
    }
    *i_bash = 0;
  }

  t = (tab_t *)Hunk_Alloc(sizeof(tab_t));
  t->name = name;
  t->type = type;

  if (!tablist)  // create list
  {
    tablist = t;
    t->next = t;
    t->prev = t;
  } else if (strcmp(name, tablist->name) < 0)  // insert at front
  {
    t->next = tablist;
    t->prev = tablist->prev;
    t->next->prev = t;
    t->prev->next = t;
    tablist = t;
  } else  // insert later
  {
    insert = tablist;
    do {
      if (strcmp(name, insert->name) < 0) break;
      insert = insert->next;
    } while (insert != tablist);

    t->next = insert;
    t->prev = insert->prev;
    t->next->prev = t;
    t->prev->next = t;
  }
}

// This is redefined from host_cmd.c
typedef struct filelist_item_s {
  char name[32];
  struct filelist_item_s *next;
} filelist_item_t;

// extern filelist_item_t *extralevels;
// extern filelist_item_t *modlist;
// extern filelist_item_t *demolist;

typedef struct arg_completion_type_s {
  const char *command;
  filelist_item_t **filelist;
} arg_completion_type_t;

static const arg_completion_type_t arg_completion_types[] = {};
//    {"map ", &extralevels},   {"changelevel ", &extralevels},
//    {"game ", &modlist},      {"record ", &demolist},
//    {"playdemo ", &demolist}, {"timedemo ", &demolist}};

static const int num_arg_completion_types =
    sizeof(arg_completion_types) / sizeof(arg_completion_types[0]);

/*
============
FindCompletion -- stevenaaus
============
*/
const char *FindCompletion(const char *partial, filelist_item_t *filelist,
                           int *nummatches_out) {
  static char matched[32];
  char *i_matched, *i_name;
  filelist_item_t *file;
  int init, match, plen;

  memset(matched, 0, sizeof(matched));
  plen = strlen(partial);
  match = 0;

  for (file = filelist, init = 0; file; file = file->next) {
    if (!strncmp(file->name, partial, plen)) {
      if (init == 0) {
        init = 1;
        strncpy(matched, file->name, sizeof(matched) - 1);
        matched[sizeof(matched) - 1] = '\0';
      } else {  // find max common
        i_matched = matched;
        i_name = file->name;
        while (*i_matched && (*i_matched == *i_name)) {
          i_matched++;
          i_name++;
        }
        *i_matched = 0;
      }
      match++;
    }
  }

  *nummatches_out = match;

  if (match > 1) {
    for (file = filelist; file; file = file->next) {
      if (!strncmp(file->name, partial, plen))
        Con_SafePrintf("   %s\n", file->name);
    }
    Con_SafePrintf("\n");
  }

  return matched;
}

/*
============
BuildTabList -- johnfitz
============
*/
void BuildTabList(const char *partial) {
  // cmdalias_t *alias;
  // cvar_t *cvar;
  // cmd_function_t *cmd;
  // int len;

  tablist = NULL;
  // len = strlen(partial);

  bash_partial[0] = 0;
  bash_singlematch = 1;

  /* TODO(therjak): repair again
  cvar = Cvar_FindVarAfter("", CVAR_NONE);
  for (; cvar; cvar = cvar->next)
    if (!Q_strncmp(partial, Cvar_GetName(cvar), len)) {
      AddToTabList(Cvar_GetName(cvar), "cvar");
    }

  for (cmd = cmd_functions; cmd; cmd = cmd->next)
    if (!Q_strncmp(partial, cmd->name, len)) AddToTabList(cmd->name, "command");


  extern cmdalias_t *cmd_alias;
  for (alias = cmd_alias; alias; alias = alias->next)
    if (!Q_strncmp(partial, alias->name, len))
      AddToTabList(alias->name, "alias");
  */
}

/*
============
Con_TabComplete -- johnfitz
============
*/
void ConTabComplete(void) {
  char partial[MAXCMDLINE];
  const char *match;
  static char *c;
  tab_t *t;
  int mark, i;

  // if editline is empty, return
  if (key_lines[edit_line][1] == 0) return;

  // get partial string (space -> cursor)
  if (!key_tabpartial[0])  // first time through, find new insert point.
                           // (Otherwise, use previous.)
  {
    // work back from cursor until you find a space, quote, semicolon, or prompt
    c = key_lines[edit_line] + key_linepos -
        1;  // start one space left of cursor
    while (*c != ' ' && *c != '\"' && *c != ';' && c != key_lines[edit_line])
      c--;
    c++;  // start 1 char after the separator we just found
  }
  for (i = 0; c + i < key_lines[edit_line] + key_linepos; i++)
    partial[i] = c[i];
  partial[i] = 0;

  // Map autocomplete function -- S.A
  // Since we don't have argument completion, this hack will do for now...
  for (i = 0; i < num_arg_completion_types; i++) {
    // arg_completion contains a command we can complete the arguments
    // for (like "map ") and a list of all the maps.
    arg_completion_type_t arg_completion = arg_completion_types[i];
    const char *command_name = arg_completion.command;

    if (!strncmp(key_lines[edit_line] + 1, command_name,
                 strlen(command_name))) {
      int nummatches = 0;
      const char *matched_map =
          FindCompletion(partial, *arg_completion.filelist, &nummatches);
      if (!*matched_map) return;
      q_strlcpy(partial, matched_map, MAXCMDLINE);
      *c = '\0';
      q_strlcat(key_lines[edit_line], partial, MAXCMDLINE);
      key_linepos = c - key_lines[edit_line] +
                    Q_strlen(matched_map);  // set new cursor position
      if (key_linepos >= MAXCMDLINE) key_linepos = MAXCMDLINE - 1;
      // if only one match, append a space
      if (key_linepos < MAXCMDLINE - 1 &&
          key_lines[edit_line][key_linepos] == 0 && (nummatches == 1)) {
        key_lines[edit_line][key_linepos] = ' ';
        key_linepos++;
        key_lines[edit_line][key_linepos] = 0;
      }
      c = key_lines[edit_line] + key_linepos;
      return;
    }
  }

  // if partial is empty, return
  if (partial[0] == 0) return;

  // trim trailing space becuase it screws up string comparisons
  if (i > 0 && partial[i - 1] == ' ') partial[i - 1] = 0;

  // find a match
  mark = Hunk_LowMark();
  if (!key_tabpartial[0])  // first time through
  {
    q_strlcpy(key_tabpartial, partial, MAXCMDLINE);
    BuildTabList(key_tabpartial);

    if (!tablist) return;

    // print list if length > 1
    if (tablist->next != tablist) {
      t = tablist;
      Con_SafePrintf("\n");
      do {
        Con_SafePrintf("   %s (%s)\n", t->name, t->type);
        t = t->next;
      } while (t != tablist);
      Con_SafePrintf("\n");
    }

    //	match = tablist->name;
    // First time, just show maximum matching chars -- S.A.
    match = bash_partial;
  } else {
    BuildTabList(key_tabpartial);

    if (!tablist) return;

    // find current match -- can't save a pointer because the list will be
    // rebuilt each time
    t = tablist;
    match = Key_ShiftDown() ? t->prev->name : t->name;
    do {
      if (!Q_strcmp(t->name, partial)) {
        match = Key_ShiftDown() ? t->prev->name : t->next->name;
        break;
      }
      t = t->next;
    } while (t != tablist);
  }
  Hunk_FreeToLowMark(mark);  // it's okay to free it here because match is a
                             // pointer to persistent data

  // insert new match into edit line
  q_strlcpy(partial, match, MAXCMDLINE);  // first copy match string
  q_strlcat(partial, key_lines[edit_line] + key_linepos,
            MAXCMDLINE);  // then add chars after cursor
  *c = '\0';              // now copy all of this into edit line
  q_strlcat(key_lines[edit_line], partial, MAXCMDLINE);
  key_linepos =
      c - key_lines[edit_line] + Q_strlen(match);  // set new cursor position
  if (key_linepos >= MAXCMDLINE) key_linepos = MAXCMDLINE - 1;

  // if cursor is at end of string, let's append a space to make life easier
  if (key_linepos < MAXCMDLINE - 1 && key_lines[edit_line][key_linepos] == 0 &&
      bash_singlematch) {
    key_lines[edit_line][key_linepos] = ' ';
    key_linepos++;
    key_lines[edit_line][key_linepos] = 0;
    // S.A.: the map argument completion (may be in combination with the
    // bash-style
    // display behavior changes, causes weirdness when completing the arguments
    // for
    // the changelevel command. the line below "fixes" it, although I'm not sure
    // about
    // the reason, yet, neither do I know any possible side effects of it:
    c = key_lines[edit_line] + key_linepos;
  }
}

/*
==============================================================================

DRAWING

==============================================================================
*/

/*
================
Con_DrawNotify

Draws the last few lines of output transparently over the game top
================
*/
void ConDrawNotify(void) {
  int i, x, v;
  const char *text;
  float time;

  GL_SetCanvas(CANVAS_CONSOLE);
  v = ConHeight();

  for (i = con_current - NUM_CON_TIMES + 1; i <= con_current; i++) {
    if (i < 0) continue;
    time = con_times[i % NUM_CON_TIMES];
    if (time == 0) continue;
    time = HostRealTime() - time;
    if (time > Cvar_GetValue(&con_notifytime)) continue;
    text = con_text + (i % con_totallines) * ConsoleWidth();

    for (x = 0; x < ConsoleWidth(); x++)
      Draw_Character((x + 1) << 3, v, text[x]);

    v += 8;

    scr_tileclear_updates = 0;
  }

  if (GetKeyDest() == key_message) {
    if (chat_team) {
      Draw_String(8, v, "say_team:");
      x = 11;
    } else {
      Draw_String(8, v, "say:");
      x = 6;
    }

    text = Key_GetChatBuffer();
    i = Key_GetChatMsgLen();
    if (i > ConsoleWidth() - x - 1) text += i - ConsoleWidth() + x + 1;

    while (*text) {
      Draw_Character(x << 3, v, *text);
      x++;
      text++;
    }

    Draw_Character(x << 3, v,
                   10 + ((int)(HostRealTime() * con_cursorspeed) & 1));
    v += 8;

    scr_tileclear_updates = 0;  // johnfitz
  }
}

/*
================
Con_DrawInput -- johnfitz -- modified to allow insert editing

The input line scrolls horizontally if typing goes beyond the right edge
================
*/
extern qpic_t *pic_ovr, *pic_ins;  // johnfitz -- new cursor handling

void Con_DrawInput(void) {
  int i, ofs;

  if (GetKeyDest() != key_console && !Con_ForceDup())
    return;  // don't draw anything

  // prestep if horizontally scrolling
  if (key_linepos >= ConsoleWidth())
    ofs = 1 + key_linepos - ConsoleWidth();
  else
    ofs = 0;

  // draw input string
  for (i = 0; key_lines[edit_line][i + ofs] && i < ConsoleWidth(); i++)
    Draw_Character((i + 1) << 3, ConHeight() - 16,
                   key_lines[edit_line][i + ofs]);

  // johnfitz -- new cursor handling
  if (!((int)((HostRealTime() - key_blinktime) * con_cursorspeed) & 1)) {
    i = key_linepos - ofs;
    Draw_Pic((i + 1) << 3, ConHeight() - 16, key_insert ? pic_ins : pic_ovr);
  }
}
