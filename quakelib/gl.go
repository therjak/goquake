package quakelib

// void GLSLGamma_GammaCorrect(void);
import "C"

type glrect struct {
	height int32
	width  int32
}

var (
	viewport glrect
)

//export GL_Height
func GL_Height() int32 {
	return viewport.height
}

//export GL_Width
func GL_Width() int32 {
	return viewport.width
}

func UpdateViewport() {
	viewport.width = int32(screen.Width)
	viewport.height = int32(screen.Height)
	statusbar.UpdateSize()
}

func GLSLGamma_GammaCorrect() {
	C.GLSLGamma_GammaCorrect()
}
