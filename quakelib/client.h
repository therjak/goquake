#ifndef _CLIENT_H_
#define _CLIENT_H_

// client_state_t should hold all pieces of the client state

#define SIGNONS 4  // signon messages to receive before connected

#include "beam.h"

#define MAX_EFRAGS 4096  // ericw -- was 2048 //johnfitz -- was 640

typedef enum {
  ca_dedicated = 0,     // a dedicated server with no ability to start a client
  ca_disconnected = 1,  // full screen console with no connection
  ca_connected = 2      // valid netcon, talking to a server
} cactive_t;

//
// the client_state_t structure is wiped completely at every
// server signon
//
typedef struct {
  // information that is static for the entire time connected to a server
  struct qmodel_s *model_precache[MAX_MODELS];

  // refresh related state
  struct qmodel_s *worldmodel;  // cl_entitites[0].model
  struct efrag_s *free_efrags;
} client_state_t;

extern entity_t cl_viewent;  // the gun model

extern client_state_t cl;

#define MAX_TEMP_ENTITIES 256    // johnfitz -- was 64
#define MAX_STATIC_ENTITIES 512  // johnfitz -- was 128
#define MAX_VISEDICTS 4096       // larger, now we support BSP2

extern entity_t cl_temp_entities[MAX_TEMP_ENTITIES];
extern entity_t *cl_visedicts[MAX_VISEDICTS];
extern int cl_numvisedicts;

//=============================================================================

// cl_main
void CL_Init(void);

// cl_input
void CL_UpdateTEnts(void);
void CL_ClearState(void);

// chase
extern cvar_t chase_active;
void Chase_Init(void);

#endif /* _CLIENT_H_ */
