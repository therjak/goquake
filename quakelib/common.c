/*
Copyright (C) 1996-2001 Id Software, Inc.
Copyright (C) 2002-2009 John Fitzgibbons and others
Copyright (C) 2010-2014 QuakeSpasm developers

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA  02111-1307, USA.

*/

// common.c -- misc functions used in client and server

#include "common.h"
#include <errno.h>
#include "q_ctype.h"
#include "quakedef.h"

const char *Com_Basedir() {
  static char buffer[MAX_OSPATH];
  char *argv = COM_BaseDir();
  strncpy(buffer, argv, MAX_OSPATH);
  free(argv);
  return buffer;
}

const char *Com_Gamedir() {
  static char buffer[MAX_OSPATH];
  char *argv = COM_GameDir();
  strncpy(buffer, argv, MAX_OSPATH);
  free(argv);
  return buffer;
}

static void COM_Path_f(void);

// if a packfile directory differs from this, it is assumed to be hacked
#define PAK0_COUNT 339      /* id1/pak0.pak - v1.0x */
#define PAK0_CRC_V100 13900 /* id1/pak0.pak - v1.00 */
#define PAK0_CRC_V101 62751 /* id1/pak0.pak - v1.01 */
#define PAK0_CRC_V106 32981 /* id1/pak0.pak - v1.06 */
#define PAK0_CRC (PAK0_CRC_V106)
#define PAK0_COUNT_V091 308 /* id1/pak0.pak - v0.91/0.92, not supported */
#define PAK0_CRC_V091 28804 /* id1/pak0.pak - v0.91/0.92, not supported */

char com_token[1024];

/*

All of Quake's data access is through a hierchal file system, but the contents
of the file system can be transparently merged from several sources.

The "base directory" is the path to the directory holding the quake.exe and all
game directories.  The sys_* files pass this to host_init in
quakeparms_t->basedir.
This can be overridden with the "-basedir" command line parm to allow code
debugging in a different directory.  The base directory is only used during
filesystem initialization.

The "game directory" is the first tree on the search path and directory that all
generated files (savegames, screenshots, demos, config files) will be saved to.
This can be overridden with the "-game" command line parameter.  The game
directory can never be changed while quake is executing.  This is a precacution
against having a malicious server instruct clients to write files over areas
they
shouldn't.

The "cache directory" is only used during development to save network bandwidth,
especially over ISDN / T1 lines.  If there is a cache directory specified, when
a file is found by the normal search path, it will be mirrored into the cache
directory, then opened there.

FIXME:
The file "parms.txt" will be read out of the game directory and appended to the
current command line arguments to allow different games to initialize startup
parms differently.  This could be used to add a "-sspeed 22050" for the high
quality sound edition.  Because they are added at the end, they will not
override an explicit setting on the original command line.

*/

//============================================================================

// ClearLink is used for new headnodes
void ClearLink(link_t *l) { l->prev = l->next = l; }

void RemoveLink(link_t *l) {
  l->next->prev = l->prev;
  l->prev->next = l->next;
}

void InsertLinkBefore(link_t *l, link_t *before) {
  l->next = before;
  l->prev = before->prev;
  l->prev->next = l;
  l->next->prev = l;
}

void InsertLinkAfter(link_t *l, link_t *after) {
  l->next = after->next;
  l->prev = after;
  l->prev->next = l;
  l->next->prev = l;
}

/*
============================================================================

                                        LIBRARY REPLACEMENT FUNCTIONS

============================================================================
*/

int q_strcasecmp(const char *s1, const char *s2) {
  const char *p1 = s1;
  const char *p2 = s2;
  char c1, c2;

  if (p1 == p2) return 0;

  do {
    c1 = q_tolower(*p1++);
    c2 = q_tolower(*p2++);
    if (c1 == '\0') break;
  } while (c1 == c2);

  return (int)(c1 - c2);
}

int q_strncasecmp(const char *s1, const char *s2, size_t n) {
  const char *p1 = s1;
  const char *p2 = s2;
  char c1, c2;

  if (p1 == p2 || n == 0) return 0;

  do {
    c1 = q_tolower(*p1++);
    c2 = q_tolower(*p2++);
    if (c1 == '\0' || c1 != c2) break;
  } while (--n > 0);

  return (int)(c1 - c2);
}

/* platform dependant (v)snprintf function names: */
#if defined(_WIN32)
#define snprintf_func _snprintf
#define vsnprintf_func _vsnprintf
#else
#define snprintf_func snprintf
#define vsnprintf_func vsnprintf
#endif

int q_vsnprintf(char *str, size_t size, const char *format, va_list args) {
  int ret;

  ret = vsnprintf_func(str, size, format, args);

  if (ret < 0) ret = (int)size;
  if (size == 0) /* no buffer */
    return ret;
  if ((size_t)ret >= size) str[size - 1] = '\0';

  return ret;
}

int q_snprintf(char *str, size_t size, const char *format, ...) {
  int ret;
  va_list argptr;

  va_start(argptr, format);
  ret = q_vsnprintf(str, size, format, argptr);
  va_end(argptr);

  return ret;
}

void Q_memset(void *dest, int fill, size_t count) {
  size_t i;

  if ((((size_t)dest | count) & 3) == 0) {
    count >>= 2;
    fill = fill | (fill << 8) | (fill << 16) | (fill << 24);
    for (i = 0; i < count; i++) ((int *)dest)[i] = fill;
  } else
    for (i = 0; i < count; i++) ((byte *)dest)[i] = fill;
}

void Q_memcpy(void *dest, const void *src, size_t count) {
  size_t i;

  if ((((size_t)dest | (size_t)src | count) & 3) == 0) {
    count >>= 2;
    for (i = 0; i < count; i++) ((int *)dest)[i] = ((int *)src)[i];
  } else
    for (i = 0; i < count; i++) ((byte *)dest)[i] = ((byte *)src)[i];
}

int Q_memcmp(const void *m1, const void *m2, size_t count) {
  while (count) {
    count--;
    if (((byte *)m1)[count] != ((byte *)m2)[count]) return -1;
  }
  return 0;
}

void Q_strcpy(char *dest, const char *src) {
  while (*src) {
    *dest++ = *src++;
  }
  *dest++ = 0;
}

int Q_strlen(const char *str) {
  int count;

  count = 0;
  while (str[count]) count++;

  return count;
}

char *Q_strrchr(const char *s, char c) {
  int len = Q_strlen(s);
  s += len;
  while (len--) {
    if (*--s == c) return (char *)s;
  }
  return NULL;
}

int Q_strcmp(const char *s1, const char *s2) {
  while (1) {
    if (*s1 != *s2) return -1;  // strings not equal
    if (!*s1) return 0;         // strings are equal
    s1++;
    s2++;
  }

  return -1;
}

int Q_strncmp(const char *s1, const char *s2, int count) {
  while (1) {
    if (!count--) return 0;
    if (*s1 != *s2) return -1;  // strings not equal
    if (!*s1) return 0;         // strings are equal
    s1++;
    s2++;
  }

  return -1;
}

int Q_atoi(const char *str) {
  int val;
  int sign;
  int c;

  if (*str == '-') {
    sign = -1;
    str++;
  } else
    sign = 1;

  val = 0;

  //
  // check for hex
  //
  if (str[0] == '0' && (str[1] == 'x' || str[1] == 'X')) {
    str += 2;
    while (1) {
      c = *str++;
      if (c >= '0' && c <= '9')
        val = (val << 4) + c - '0';
      else if (c >= 'a' && c <= 'f')
        val = (val << 4) + c - 'a' + 10;
      else if (c >= 'A' && c <= 'F')
        val = (val << 4) + c - 'A' + 10;
      else
        return val * sign;
    }
  }

  //
  // check for character
  //
  if (str[0] == '\'') {
    return sign * str[1];
  }

  //
  // assume decimal
  //
  while (1) {
    c = *str++;
    if (c < '0' || c > '9') return val * sign;
    val = val * 10 + c - '0';
  }

  return 0;
}

/*
============
COM_SkipPath
============
*/

const char *COM_SkipPath(const char *pathname) {
  const char *last;

  last = pathname;
  while (*pathname) {
    if (*pathname == '/') last = pathname + 1;
    pathname++;
  }
  return last;
}

/*
============
COM_StripExtension
============
*/
void COM_StripExtension(const char *in, char *out, size_t outsize) {
  int length;

  if (!*in) {
    *out = '\0';
    return;
  }
  if (in != out) /* copy when not in-place editing */
    q_strlcpy(out, in, outsize);
  length = (int)strlen(out) - 1;
  while (length > 0 && out[length] != '.') {
    --length;
    if (out[length] == '/' || out[length] == '\\') return; /* no extension */
  }
  if (length > 0) out[length] = '\0';
}

/*
============
COM_FileGetExtension - doesn't return NULL
============
*/
const char *COM_FileGetExtension(const char *in) {
  const char *src;
  size_t len;

  len = strlen(in);
  if (len < 2) /* nothing meaningful */
    return "";

  src = in + len - 1;
  while (src != in && src[-1] != '.') src--;
  if (src == in || strchr(src, '/') != NULL || strchr(src, '\\') != NULL)
    return ""; /* no extension, or parent directory has a dot */

  return src;
}

/*
============
COM_FileBase
take 'somedir/otherdir/filename.ext',
write only 'filename' to the output
============
*/
void COM_FileBase(const char *in, char *out, size_t outsize) {
  const char *dot, *slash, *s;

  s = in;
  slash = in;
  dot = NULL;
  while (*s) {
    if (*s == '/') slash = s + 1;
    if (*s == '.') dot = s;
    s++;
  }
  if (dot == NULL) dot = s;

  if (dot - slash < 2)
    q_strlcpy(out, "?model?", outsize);
  else {
    size_t len = dot - slash;
    if (len >= outsize) len = outsize - 1;
    memcpy(out, slash, len);
    out[len] = '\0';
  }
}

/*
==================
COM_AddExtension
if path extension doesn't match .EXT, append it
(extension should include the leading ".")
==================
*/
void COM_AddExtension(char *path, const char *extension, size_t len) {
  if (strcmp(COM_FileGetExtension(path), extension + 1) != 0)
    q_strlcat(path, extension, len);
}

/*
==============
COM_Parse

Parse a token out of a string
==============
*/
const char *COM_Parse(const char *data) {
  int c;
  int len;

  len = 0;
  com_token[0] = 0;

  if (!data) return NULL;

// skip whitespace
skipwhite:
  while ((c = *data) <= ' ') {
    if (c == 0) return NULL;  // end of file
    data++;
  }

  // skip // comments
  if (c == '/' && data[1] == '/') {
    while (*data && *data != '\n') data++;
    goto skipwhite;
  }

  // skip /*..*/ comments
  if (c == '/' && data[1] == '*') {
    data += 2;
    while (*data && !(*data == '*' && data[1] == '/')) data++;
    if (*data) data += 2;
    goto skipwhite;
  }

  // handle quoted strings specially
  if (c == '\"') {
    data++;
    while (1) {
      if ((c = *data) != 0) ++data;
      if (c == '\"' || !c) {
        com_token[len] = 0;
        return data;
      }
      com_token[len] = c;
      len++;
    }
  }

  // parse single characters
  if (c == '{' || c == '}' || c == '(' || c == ')' || c == '\'' || c == ':') {
    com_token[len] = c;
    len++;
    com_token[len] = 0;
    return data + 1;
  }

  // parse a regular word
  do {
    com_token[len] = c;
    data++;
    len++;
    c = *data;
    /* commented out the check for ':' so that ip:port works */
    if (c == '{' || c == '}' || c == '(' || c == ')' ||
        c == '\'' /* || c == ':' */)
      break;
  } while (c > 32);

  com_token[len] = 0;
  return data;
}

/*
============
va

does a varargs printf into a temp buffer. cycles between
4 different static buffers. the number of buffers cycled
is defined in VA_NUM_BUFFS.
FIXME: make this buffer size safe someday
============
*/
#define VA_NUM_BUFFS 4
#define VA_BUFFERLEN 1024

static char *get_va_buffer(void) {
  static char va_buffers[VA_NUM_BUFFS][VA_BUFFERLEN];
  static int buffer_idx = 0;
  buffer_idx = (buffer_idx + 1) & (VA_NUM_BUFFS - 1);
  return va_buffers[buffer_idx];
}

char *va(const char *format, ...) {
  va_list argptr;
  char *va_buf;

  va_buf = get_va_buffer();
  va_start(argptr, format);
  q_vsnprintf(va_buf, VA_BUFFERLEN, format, argptr);
  va_end(argptr);

  return va_buf;
}

/*
=============================================================================

QUAKE FILESYSTEM

=============================================================================
*/

/*
============
COM_CreatePath
============
*/
void COM_CreatePath(char *path) {
  char *ofs;

  for (ofs = path + 1; *ofs; ofs++) {
    if (*ofs == '/') {  // create the directory
      *ofs = 0;
      Sys_mkdir(path);
      *ofs = '/';
    }
  }
}

//==============================================================================
// johnfitz -- dynamic gamedir stuff -- modified by QuakeSpasm team.
//==============================================================================
void ExtraMaps_NewGame(void);
static void COM_Game_f(void) {
  // TODO(therjak): broken as Cmd_Argv point always to the same buffer
}
