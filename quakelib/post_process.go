package quakelib

func postProcessGammaContrast(gamma, contrast float32) {
	GLSLGamma_GammaCorrect()
}
