#ifndef _QUAKE_SYS_H
#define _QUAKE_SYS_H

int Sys_FileTime(const char *path);
void Sys_mkdir(const char *path);
void Sys_Error(const char *error, ...);
const char *Sys_ConsoleInput(void);

#endif /* _QUAKE_SYS_H */
