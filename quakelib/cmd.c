#include "quakedef.h"

const char* Cmd_Argv(int arg) {
  static char buffer[2048];
  char* argv = Cmd_ArgvInt(arg);
  strncpy(buffer, argv, 2048);
  free(argv);
  return buffer;
}

void callQuakeFunc(xcommand_t f) { f(); }

void setInt(int* l, int v) { *l = v; }

