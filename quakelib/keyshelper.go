package quakelib

//#include <stdlib.h>
// void M_FindKeysForCommand(const char* command, int* threekeys);
// void M_UnbindCommand(const char* command);
// typedef struct {
// int a;
// int b;
// int c;
// } keyTrip;
// keyTrip go_findKeys(const char* command) {
//   keyTrip kt;
//   int k[3];
//   M_FindKeysForCommand(command, k);
//   kt.a = k[0];
//   kt.b = k[1];
//   kt.c = k[2];
//   return kt;
// }
import "C"
import "unsafe"

func getKeysForCommand(c string) (int, int, int) {
	cn := C.CString(c)
	defer C.free(unsafe.Pointer(cn))
	k := C.go_findKeys(cn)
	return int(k.a), int(k.b), int(k.c)
}

func unbindCommand(c string) {
	cn := C.CString(c)
	defer C.free(unsafe.Pointer(cn))
	C.M_UnbindCommand(cn)
}
