#include "quakedef.h"

void SwapPic(qpic_t *pic) {
  pic->width = LittleLong(pic->width);
  pic->height = LittleLong(pic->height);
}
