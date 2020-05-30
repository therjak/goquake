package quakelib

//void Sky_Init(void);
//void Sky_DrawSky(void);
//void Sky_NewMap(void);
//void Sky_LoadSkyBox(const char *name);
//void Sky_LoadTextureInt(const unsigned char* src, const char* skyName, const char* modelName);
import "C"

//export SkyInit
func SkyInit() {
	C.Sky_Init()
}

//export SkyDrawSky
func SkyDrawSky() {
	C.Sky_DrawSky()
}

//export SkyNewMap
func SkyNewMap() {
	C.Sky_NewMap()
}

//export SkyLoadSkyBox
func SkyLoadSkyBox(c *C.char) {
	C.Sky_LoadSkyBox(c)
}

//export SkyLoadTexture
func SkyLoadTexture(src *C.uchar, skyName *C.char, modelName *C.char) {
	C.Sky_LoadTextureInt(src, skyName, modelName)
}
