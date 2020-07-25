package quakelib

// void GLSLGamma_GammaCorrect(void);
// void GLSLGamma_DeleteTexture(void);
import "C"

//export GL_Height
func GL_Height() int {
	return screen.Height
}

//export GL_Width
func GL_Width() int {
	return screen.Width
}

func GLSLGamma_GammaCorrect() {
	C.GLSLGamma_GammaCorrect()
}

func GLSLGamma_DeleteTexture() {
	C.GLSLGamma_DeleteTexture()
}
