package quakelib

//void Sbar_Changed();
import "C"

func StatusbarChanged() {
  C.Sbar_Changed()
}
