// zone.c

#include "quakedef.h"

#define DYNAMIC_SIZE \
  (4 * 1024 * 1024)  // ericw -- was 512KB (64-bit) / 384KB (32-bit)

#define ZONEID 0x1d4a11
#define MINFRAGMENT 64

void Cache_Free(cache_user_t *c, qboolean freetextures);

typedef struct memblock_s {
  int size;  // including the header and possibly tiny fragments
  int tag;   // a tag of 0 is a free block
  int id;    // should be ZONEID
  int pad;   // pad to 64 bit boundary
  struct memblock_s *next, *prev;
} memblock_t;

typedef struct {
  int size;              // total bytes malloced, including header
  memblock_t blocklist;  // start / end cap for linked list
  memblock_t *rover;
} memzone_t;

void Cache_FreeLow(int new_low_hunk);
void Cache_FreeHigh(int new_high_hunk);

/*
==============================================================================

                                                ZONE MEMORY ALLOCATION

There is never any space between memblocks, and there will never be two
contiguous free memblocks.

The rover can be left pointing at a non-empty block

The zone calls are pretty much only used for small strings and structures,
all big things are allocated on the hunk.
==============================================================================
*/

static memzone_t *mainzone;

/*
========================
Z_Free
========================
*/
void Z_Free(void *ptr) {
  memblock_t *block, *other;

  if (!ptr) Go_Error("Z_Free: NULL pointer");

  block = (memblock_t *)((byte *)ptr - sizeof(memblock_t));
  if (block->id != ZONEID) Go_Error("Z_Free: freed a pointer without ZONEID");
  if (block->tag == 0) Go_Error("Z_Free: freed a freed pointer");

  block->tag = 0;  // mark as free

  other = block->prev;
  if (!other->tag) {  // merge with previous free block
    other->size += block->size;
    other->next = block->next;
    other->next->prev = other;
    if (block == mainzone->rover) mainzone->rover = other;
    block = other;
  }

  other = block->next;
  if (!other->tag) {  // merge the next free block onto the end
    block->size += other->size;
    block->next = other->next;
    block->next->prev = block;
    if (other == mainzone->rover) mainzone->rover = block;
  }
}

static void *Z_TagMalloc(int size, int tag) {
  int extra;
  memblock_t *start, *rover, *newblock, *base;

  if (!tag) Go_Error("Z_TagMalloc: tried to use a 0 tag");

  //
  // scan through the block list looking for the first free block
  // of sufficient size
  //
  size += sizeof(memblock_t);  // account for size of block header
  size += 4;                   // space for memory trash tester
  size = (size + 7) & ~7;      // align to 8-byte boundary

  base = rover = mainzone->rover;
  start = base->prev;

  do {
    if (rover == start)  // scaned all the way around the list
      return NULL;
    if (rover->tag)
      base = rover = rover->next;
    else
      rover = rover->next;
  } while (base->tag || base->size < size);

  //
  // found a block big enough
  //
  extra = base->size - size;
  if (extra >
      MINFRAGMENT) {  // there will be a free fragment after the allocated block
    newblock = (memblock_t *)((byte *)base + size);
    newblock->size = extra;
    newblock->tag = 0;  // free block
    newblock->prev = base;
    newblock->id = ZONEID;
    newblock->next = base->next;
    newblock->next->prev = newblock;
    base->next = newblock;
    base->size = size;
  }

  base->tag = tag;  // no longer a free block

  mainzone->rover = base->next;  // next allocation will start looking here

  base->id = ZONEID;

  // marker for memory trash testing
  *(int *)((byte *)base + base->size - 4) = ZONEID;

  return (void *)((byte *)base + sizeof(memblock_t));
}

/*
========================
Z_CheckHeap
========================
*/
static void Z_CheckHeap(void) {
  memblock_t *block;

  for (block = mainzone->blocklist.next;; block = block->next) {
    if (block->next == &mainzone->blocklist) break;  // all blocks have been hit
    if ((byte *)block + block->size != (byte *)block->next)
      Go_Error("Z_CheckHeap: block size does not touch the next block\n");
    if (block->next->prev != block)
      Go_Error("Z_CheckHeap: next block doesn't have proper back link\n");
    if (!block->tag && !block->next->tag)
      Go_Error("Z_CheckHeap: two consecutive free blocks\n");
  }
}

/*
========================
Z_Malloc
========================
*/
void *Z_Malloc(int size) {
  void *buf;

  Z_CheckHeap();  // DEBUG
  buf = Z_TagMalloc(size, 1);
  if (!buf) Go_Error_I("Z_Malloc: failed on allocation of %v bytes", size);
  Q_memset(buf, 0, size);

  return buf;
}

char *Z_Strdup(const char *s) {
  size_t sz = strlen(s) + 1;
  char *ptr = (char *)Z_Malloc(sz);
  memcpy(ptr, s, sz);
  return ptr;
}

//============================================================================

#define HUNK_SENTINAL 0x1df001ed

#define HUNKNAME_LEN 24
typedef struct {
  int sentinal;
  int size;  // including sizeof(hunk_t), -1 = not allocated
  char name[HUNKNAME_LEN];
} hunk_t;

byte *hunk_base;
int hunk_size;

int hunk_low_used;
int hunk_high_used;

qboolean hunk_tempactive;
int hunk_tempmark;

/*
==============
Hunk_Check

Run consistancy and sentinal trahing checks
==============
*/
void Hunk_Check(void) {
  hunk_t *h;

  for (h = (hunk_t *)hunk_base; (byte *)h != hunk_base + hunk_low_used;) {
    if (h->sentinal != HUNK_SENTINAL) Go_Error("Hunk_Check: trahsed sentinal");
    if (h->size < (int)sizeof(hunk_t) ||
        h->size + (byte *)h - hunk_base > hunk_size)
      Go_Error("Hunk_Check: bad size");
    h = (hunk_t *)((byte *)h + h->size);
  }
}

/*
===================
Hunk_AllocName
===================
*/
void *Hunk_AllocName(int size, const char *name) {
  hunk_t *h;

#ifdef PARANOID
  Hunk_Check();
#endif

  if (size < 0) Go_Error_I("Hunk_Alloc: bad size: %v", size);

  size = sizeof(hunk_t) + ((size + 15) & ~15);

  if (hunk_size - hunk_low_used - hunk_high_used < size)
    Go_Error_I("Hunk_Alloc: failed on %v bytes", size);

  h = (hunk_t *)(hunk_base + hunk_low_used);
  hunk_low_used += size;

  Cache_FreeLow(hunk_low_used);

  memset(h, 0, size);

  h->size = size;
  h->sentinal = HUNK_SENTINAL;
  q_strlcpy(h->name, name, HUNKNAME_LEN);

  return (void *)(h + 1);
}

/*
===================
Hunk_Alloc
===================
*/
void *Hunk_Alloc(int size) { return Hunk_AllocName(size, "unknown"); }

int Hunk_LowMark(void) { return hunk_low_used; }

void Hunk_FreeToLowMark(int mark) {
  if (mark < 0 || mark > hunk_low_used)
    Go_Error_I("Hunk_FreeToLowMark: bad mark %v", mark);
  memset(hunk_base + mark, 0, hunk_low_used - mark);
  hunk_low_used = mark;
}

/*
===============================================================================

CACHE MEMORY

===============================================================================
*/

#define CACHENAME_LEN 32
typedef struct cache_system_s {
  int size;  // including this header
  cache_user_t *user;
  char name[CACHENAME_LEN];
  struct cache_system_s *prev, *next;
  struct cache_system_s *lru_prev, *lru_next;  // for LRU flushing
} cache_system_t;

cache_system_t *Cache_TryAlloc(int size, qboolean nobottom);

cache_system_t cache_head;

/*
===========
Cache_Move
===========
*/
void Cache_Move(cache_system_t *c) {
  cache_system_t *new_cs;

  // we are clearing up space at the bottom, so only allocate it late
  new_cs = Cache_TryAlloc(c->size, true);
  if (new_cs) {
    Q_memcpy(new_cs + 1, c + 1, c->size - sizeof(cache_system_t));
    new_cs->user = c->user;
    Q_memcpy(new_cs->name, c->name, sizeof(new_cs->name));
    Cache_Free(c->user, false);  // johnfitz -- added second argument
    new_cs->user->data = (void *)(new_cs + 1);
  } else {
    Cache_Free(c->user,
               true);  // tough luck... //johnfitz -- added second argument
  }
}

/*
============
Cache_FreeLow

Throw things out until the hunk can be expanded to the given point
============
*/
void Cache_FreeLow(int new_low_hunk) {
  cache_system_t *c;

  while (1) {
    c = cache_head.next;
    if (c == &cache_head) return;  // nothing in cache at all
    if ((byte *)c >= hunk_base + new_low_hunk)
      return;       // there is space to grow the hunk
    Cache_Move(c);  // reclaim the space
  }
}

/*
============
Cache_FreeHigh

Throw things out until the hunk can be expanded to the given point
============
*/
void Cache_FreeHigh(int new_high_hunk) {
  cache_system_t *c, *prev;

  prev = NULL;
  while (1) {
    c = cache_head.prev;
    if (c == &cache_head) return;  // nothing in cache at all
    if ((byte *)c + c->size <= hunk_base + hunk_size - new_high_hunk)
      return;  // there is space to grow the hunk
    if (c == prev)
      Cache_Free(c->user, true);  // didn't move out of the way //johnfitz --
                                  // added second argument
    else {
      Cache_Move(c);  // try to move it
      prev = c;
    }
  }
}

void Cache_UnlinkLRU(cache_system_t *cs) {
  if (!cs->lru_next || !cs->lru_prev) Go_Error("Cache_UnlinkLRU: NULL link");

  cs->lru_next->lru_prev = cs->lru_prev;
  cs->lru_prev->lru_next = cs->lru_next;

  cs->lru_prev = cs->lru_next = NULL;
}

void Cache_MakeLRU(cache_system_t *cs) {
  if (cs->lru_next || cs->lru_prev) Go_Error("Cache_MakeLRU: active link");

  cache_head.lru_next->lru_prev = cs;
  cs->lru_next = cache_head.lru_next;
  cs->lru_prev = &cache_head;
  cache_head.lru_next = cs;
}

/*
============
Cache_TryAlloc

Looks for a free block of memory between the high and low hunk marks
Size should already include the header and padding
============
*/
cache_system_t *Cache_TryAlloc(int size, qboolean nobottom) {
  cache_system_t *cs, *new_cs;

  // is the cache completely empty?

  if (!nobottom && cache_head.prev == &cache_head) {
    if (hunk_size - hunk_high_used - hunk_low_used < size)
      Go_Error_I("Cache_TryAlloc: %v is greater then free hunk", size);

    new_cs = (cache_system_t *)(hunk_base + hunk_low_used);
    memset(new_cs, 0, sizeof(*new_cs));
    new_cs->size = size;

    cache_head.prev = cache_head.next = new_cs;
    new_cs->prev = new_cs->next = &cache_head;

    Cache_MakeLRU(new_cs);
    return new_cs;
  }

  // search from the bottom up for space

  new_cs = (cache_system_t *)(hunk_base + hunk_low_used);
  cs = cache_head.next;

  do {
    if (!nobottom || cs != cache_head.next) {
      if ((byte *)cs - (byte *)new_cs >= size) {  // found space
        memset(new_cs, 0, sizeof(*new_cs));
        new_cs->size = size;

        new_cs->next = cs;
        new_cs->prev = cs->prev;
        cs->prev->next = new_cs;
        cs->prev = new_cs;

        Cache_MakeLRU(new_cs);

        return new_cs;
      }
    }

    // continue looking
    new_cs = (cache_system_t *)((byte *)cs + cs->size);
    cs = cs->next;

  } while (cs != &cache_head);

  // try to allocate one at the very end
  if (hunk_base + hunk_size - hunk_high_used - (byte *)new_cs >= size) {
    memset(new_cs, 0, sizeof(*new_cs));
    new_cs->size = size;

    new_cs->next = &cache_head;
    new_cs->prev = cache_head.prev;
    cache_head.prev->next = new_cs;
    cache_head.prev = new_cs;

    Cache_MakeLRU(new_cs);

    return new_cs;
  }

  return NULL;  // couldn't allocate
}

/*
============
Cache_Init

============
*/
void Cache_Init(void) {
  cache_head.next = cache_head.prev = &cache_head;
  cache_head.lru_next = cache_head.lru_prev = &cache_head;
}

/*
==============
Cache_Free

Frees the memory and removes it from the LRU list
==============
*/
void Cache_Free(cache_user_t *c,
                qboolean freetextures)  // johnfitz -- added second argument
{
  cache_system_t *cs;

  if (!c->data) Go_Error("Cache_Free: not allocated");

  cs = ((cache_system_t *)c->data) - 1;

  cs->prev->next = cs->next;
  cs->next->prev = cs->prev;
  cs->next = cs->prev = NULL;

  c->data = NULL;

  Cache_UnlinkLRU(cs);

  // johnfitz -- if a model becomes uncached, free the gltextures.  This only
  // works
  // becuase the cache_user_t is the last component of the qmodel_t struct.
  // Should
  // fail harmlessly if *c is actually part of an sfx_t struct.  I FEEL DIRTY
  if (freetextures) TexMgr_FreeTexturesForOwner((qmodel_t *)(c + 1) - 1);
}

/*
==============
Cache_Check
==============
*/
void *Cache_Check(cache_user_t *c) {
  cache_system_t *cs;

  if (!c->data) return NULL;

  cs = ((cache_system_t *)c->data) - 1;

  // move to head of LRU
  Cache_UnlinkLRU(cs);
  Cache_MakeLRU(cs);

  return c->data;
}

/*
==============
Cache_Alloc
==============
*/
void *Cache_Alloc(cache_user_t *c, int size, const char *name) {
  cache_system_t *cs;

  if (c->data) Go_Error("Cache_Alloc: allready allocated");

  if (size <= 0) Go_Error_I("Cache_Alloc: size %v", size);

  size = (size + sizeof(cache_system_t) + 15) & ~15;

  // find memory for it
  while (1) {
    cs = Cache_TryAlloc(size, false);
    if (cs) {
      q_strlcpy(cs->name, name, CACHENAME_LEN);
      c->data = (void *)(cs + 1);
      cs->user = c;
      break;
    }

    // free the least recently used cahedat
    if (cache_head.lru_prev == &cache_head)
      Go_Error("Cache_Alloc: out of memory");  // not enough memory at all

    Cache_Free(cache_head.lru_prev->user,
               true);  // johnfitz -- added second argument
  }

  return Cache_Check(c);
}

//============================================================================

static void Memory_InitZone(memzone_t *zone, int size) {
  memblock_t *block;

  // set the entire zone to one free block

  zone->blocklist.next = zone->blocklist.prev = block =
      (memblock_t *)((byte *)zone + sizeof(memzone_t));
  zone->blocklist.tag = 1;  // in use block
  zone->blocklist.id = 0;
  zone->blocklist.size = 0;
  zone->rover = block;

  block->prev = block->next = &zone->blocklist;
  block->tag = 0;  // free block
  block->id = ZONEID;
  block->size = size - sizeof(memzone_t);
}

/*
========================
Memory_Init
========================
*/
void Memory_Init(void *buf, int size) {
  int p;
  int zonesize = CMLZone();

  hunk_base = (byte *)buf;
  hunk_size = size;
  hunk_low_used = 0;
  hunk_high_used = 0;

  Cache_Init();
  mainzone = (memzone_t *)Hunk_AllocName(zonesize, "zone");
  Memory_InitZone(mainzone, zonesize);
}
