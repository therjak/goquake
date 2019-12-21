#ifndef _QUAKE_SERVER_H
#define _QUAKE_SERVER_H

const char *SV_ModelName();

// edict->solid values
#define SOLID_NOT 0       // no interaction with other objects
#define SOLID_TRIGGER 1   // touch on edge, but not blocking
#define SOLID_BBOX 2      // touch on edge, block
#define SOLID_SLIDEBOX 3  // touch on edge, but not an onground
#define SOLID_BSP 4       // bsp clip, touch on edge, block

// entity effects

#define EF_BRIGHTFIELD 1
#define EF_MUZZLEFLASH 2
#define EF_BRIGHTLIGHT 4
#define EF_DIMLIGHT 8

#endif /* _QUAKE_SERVER_H */
