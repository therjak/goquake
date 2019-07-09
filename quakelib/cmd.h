#ifndef _QUAKE_CMD_H
#define _QUAKE_CMD_H

typedef enum {
  src_client,  // came in over a net connection as a clc_stringcmd
               // host_client will be valid during this state.
  src_command  // from the command buffer
} cmd_source_t;

const char *Cmd_Argv(int arg);
// The functions that execute commands get their parameters with these
// functions. Cmd_Argv () will return an empty string, not a NULL
// if arg > argc, so string operations are allways safe.

#endif /* _QUAKE_CMD_H */
