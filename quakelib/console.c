// SPDX-License-Identifier: GPL-2.0-or-later
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
