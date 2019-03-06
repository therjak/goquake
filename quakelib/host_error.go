package quakelib

//#include <string.h>
//void Host_Error(const char * error, ...);
//void host_error_go(char* error) {
//  char string[1024];
//  strncpy(string, error, 1024);
//  free(error);
//  Host_Error(string);
//}
import "C"
import "fmt"

func HostError(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	C.host_error_go(C.CString(s))
}
