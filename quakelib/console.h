#ifndef __CONSOLE_H
#define __CONSOLE_H

void Con_Printf(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));
void Con_DWarning(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));  // ericw
void Con_Warning(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));  // johnfitz
void Con_DPrintf(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));
void Con_DPrintf2(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));  // johnfitz
void Con_SafePrintf(const char *fmt, ...)
    __attribute__((__format__(__printf__, 1, 2)));

void Sys_Error(const char *error, ...);
#endif /* __CONSOLE_H */
