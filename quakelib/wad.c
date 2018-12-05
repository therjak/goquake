// wad.c

#include "quakedef.h"

typedef struct {
  char identification[4];  // should be WAD2 or 2DAW
  int numlumps;
  int infotableofs;
} wadinfo_t;

typedef struct {
  int filepos;
  int disksize;
  int size;  // uncompressed
  char type;
  char compression;
  char pad1, pad2;
  char name[16];  // must be null terminated
} lumpinfo_t;

int wad_numlumps;
lumpinfo_t *wad_lumps;
byte *wad_base = NULL;

void SwapPic(qpic_t *pic);

/*
==================
W_CleanupName

Lowercases name and pads with spaces and a terminating 0 to the length of
lumpinfo_t->name.
Used so lumpname lookups can proceed rapidly by comparing 4 chars at a time
Space padding is so names can be printed nicely in tables.
Can safely be performed in place.
==================
*/
void W_CleanupName(const char *in, char *out) {
  int i;
  int c;

  for (i = 0; i < 16; i++) {
    c = in[i];
    if (!c) break;

    if (c >= 'A' && c <= 'Z') c += ('a' - 'A');
    out[i] = c;
  }

  for (; i < 16; i++) out[i] = 0;
}

/*
====================
W_LoadWadFile
====================
*/
void W_LoadWadFile(void)  // johnfitz -- filename is now hard-coded for honesty
{
  lumpinfo_t *lump_p;
  wadinfo_t *header;
  int i;
  int infotableofs;
  const char *filename = WADFILENAME;

  // johnfitz -- modified to use malloc
  // TODO: use cache_alloc
  if (wad_base) free(wad_base);
  int length = 0;
  wad_base = COM_LoadFileGo(filename, &length);
  if (!wad_base) Go_Error_S("W_LoadWadFile: couldn't load %v", filename);

  header = (wadinfo_t *)wad_base;

  if (header->identification[0] != 'W' || header->identification[1] != 'A' ||
      header->identification[2] != 'D' || header->identification[3] != '2')
    Go_Error_S("Wad file %v doesn't have WAD2 id\n", filename);

  wad_numlumps = LittleLong(header->numlumps);
  infotableofs = LittleLong(header->infotableofs);
  wad_lumps = (lumpinfo_t *)(wad_base + infotableofs);

  for (i = 0, lump_p = wad_lumps; i < wad_numlumps; i++, lump_p++) {
    lump_p->filepos = LittleLong(lump_p->filepos);
    lump_p->size = LittleLong(lump_p->size);
    W_CleanupName(lump_p->name, lump_p->name);  // CAUTION: in-place editing!!!
    if (lump_p->type == 66)                     // 66 == TYP_QPIC
      SwapPic((qpic_t *)(wad_base + lump_p->filepos));
  }
}

/*
=============
W_GetLumpinfo
=============
*/
lumpinfo_t *W_GetLumpinfo(const char *name) {
  int i;
  lumpinfo_t *lump_p;
  char clean[16];

  W_CleanupName(name, clean);

  for (lump_p = wad_lumps, i = 0; i < wad_numlumps; i++, lump_p++) {
    if (!strcmp(clean, lump_p->name)) return lump_p;
  }

  Con_SafePrintf("W_GetLumpinfo: %s not found\n",
                 name);  // johnfitz -- was Sys_Error
  return NULL;
}

void *W_GetLumpName(const char *name) {
  lumpinfo_t *lump;

  lump = W_GetLumpinfo(name);

  if (!lump) return NULL;  // johnfitz

  return (void *)(wad_base + lump->filepos);
}

qpic_t *W_GetQPic(const char *name) { return (qpic_t *)W_GetLumpName(name); }

byte *W_GetConchars() { return (byte *)W_GetLumpName("conchars"); }

/*
=============================================================================

automatic byte swapping

=============================================================================
*/

void SwapPic(qpic_t *pic) {
  pic->width = LittleLong(pic->width);
  pic->height = LittleLong(pic->height);
}
