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

extern int history_line;  // johnfitz

/*
================
Con_Init
================
*/
void ConInit(void) {
  SetConsoleWidth(38);
  Con_Printf("Console initialized.\n");
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
