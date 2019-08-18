#ifndef _QUAKE_ENTITY_STATE_H
#define _QUAKE_ENTITY_STATE_H

typedef struct {
  float origin[3];
  float angles[3];
  unsigned short modelindex;  // johnfitz -- was int
  unsigned short frame;       // johnfitz -- was int
  unsigned char skin;         // johnfitz -- was int
  unsigned char alpha;        // johnfitz -- added
  int effects;
} entity_state_t;

#endif  // _QUAKE_ENTITY_STATE_H
