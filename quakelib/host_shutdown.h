void Host_Shutdown(void);
void Con_PrintStr(const char* text);
struct cvar_s;
typedef void (*cvarcallback_t)(struct cvar_s*);
void CallCvarCallback(int id, cvarcallback_t func);
