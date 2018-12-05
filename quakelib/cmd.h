#ifndef _QUAKE_CMD_H
#define _QUAKE_CMD_H

typedef enum {
  src_client,  // came in over a net connection as a clc_stringcmd
               // host_client will be valid during this state.
  src_command  // from the command buffer
} cmd_source_t;

const char *Cmd_Argv(int arg);
const char *Cmd_Args(void);
// The functions that execute commands get their parameters with these
// functions. Cmd_Argv () will return an empty string, not a NULL
// if arg > argc, so string operations are allways safe.

void Cmd_ForwardToServer(void);
// adds the current command line as a clc_stringcmd to the client message.
// things like godmode, noclip, etc, are commands directed to the server,
// so when they are typed in at the console, they will need to be forwarded.

#endif /* _QUAKE_CMD_H */
