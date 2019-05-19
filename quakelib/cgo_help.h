#ifndef CFWRAP
#define CFWRAP
inline float cf(int idx, float* f) { return f[idx]; }
inline float* cfp(int idx, float* f) { return &f[idx]; }
#endif
