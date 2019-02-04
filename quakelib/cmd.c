#include "quakedef.h"

const char* Cmd_Argv(int arg) {
  static char buffer[2048];
  char* argv = Cmd_ArgvInt(arg);
  strncpy(buffer, argv, 2048);
  free(argv);
  return buffer;
}

const char* Cmd_Args(void) {
  static char buffer[2048];
  char* argv = Cmd_ArgsInt();
  strncpy(buffer, argv, 2048);
  free(argv);
  return buffer;
}

void callQuakeFunc(xcommand_t f) { f(); }

void setInt(int* l, int v) { *l = v; }

/*
===================
Cmd_ForwardToServer

Sends the entire command line over to the server
===================
*/

void Cmd_ForwardToServer(void) {
  if (CLS_GetState() != ca_connected) {
    Con_Printf("Can't \"%s\", not connected\n", Cmd_Argv(0));
    return;
  }

  if (CLS_IsDemoPlayback()) return;  // not really connected

  CLSMessageWriteByte(clc_stringcmd);
  if (q_strcasecmp(Cmd_Argv(0), "cmd") != 0) {
    CLSMessagePrint(Cmd_Argv(0));
    CLSMessagePrint(" ");
  }
  if (Cmd_Argc() > 1) {
    CLSMessagePrint(Cmd_Args());
  } else {
    CLSMessagePrint("\n");
  }
  // CLSMessagePrint was previously overriding a trailing 0 if it existed.
  // As this is the only place CLSMessagePrint is used just never add a
  // trailing 0 and add the wanted one explicit. CLSMessageWriteByte does
  // not add an additional 0, so the first CLSMessagePrint does not find
  // a trailing 0.
  CLSMessageWriteByte(0);
}
