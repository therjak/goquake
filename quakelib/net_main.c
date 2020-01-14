#include "_cgo_export.h"

const char *CL_MSG_ReadString(void) {
  static char string[2048];
  int c;
  size_t l;

  l = 0;
  do {
    c = CL_MSG_ReadByte();
    if (c == -1 || c == 0) break;
    string[l] = c;
    l++;
  } while (l < sizeof(string) - 1);

  string[l] = 0;
  return string;
}
