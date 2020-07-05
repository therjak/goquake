package commandline

import (
	"flag"
	"fmt"
	"strconv"
)

var (
	cocombine  bool
	conDebug   bool
	current    bool
	f          bool
	fitz       bool
	fullscreen bool
	hipnotic   bool
	minMemory  bool
	noMouse    bool
	noSound    bool
	quoth      bool
	rogue      bool
	safeMode   bool
	w          bool
	window     bool

	dedicated = boolInt{false, 8}
	listen    = boolInt{false, 8}

	bpp       int
	conSize   int
	fsaa      int
	height    int
	particles int
	port      int
	protocol  int
	width     int
	zone      int

	basedir string
	game    string
)

type boolInt struct {
	set bool
	num int
}

func (b *boolInt) IsBoolFlag() bool {
	// We can not support both "-flag" and "-flag 10"
	// This allows "-flag", and "-flag=10"
	// and also "-flag=true" and "-flag=false"
	// but not "-flag 10"
	return true
}

func (b *boolInt) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	if err != nil {
		v, err := strconv.ParseBool(s)
		b.set = v
		return err
	}
	b.set = true
	b.num = int(v)
	return nil
}

func (b *boolInt) String() string {
	return fmt.Sprintf("Set: %v, Num: %v", b.set, b.num)
}

func init() {
	flag.BoolVar(&conDebug, "condebug", false, "enable console debugging")
	flag.BoolVar(&current, "current", false, "Runs as dedicated server")
	flag.BoolVar(&fitz, "fitz", false, "")
	flag.BoolVar(&fullscreen, "f", false, "")
	flag.BoolVar(&fullscreen, "fullscreen", false, "")
	flag.BoolVar(&hipnotic, "hipnotic", false, "Runs as dedicated server")
	flag.BoolVar(&minMemory, "minmemory", false, "")
	flag.BoolVar(&noMouse, "nomouse", false, "Disable mouse input")
	flag.BoolVar(&noSound, "nosound", false, "Disable sound output")
	flag.BoolVar(&quoth, "quoth", false, "Runs quoth")
	flag.BoolVar(&rogue, "roque", false, "Runs rogue")
	// TODO: safe should enable noMouse and noSound
	flag.BoolVar(&safeMode, "safe", false, "Runs in safe mode")
	flag.BoolVar(&window, "window", false, "")
	flag.BoolVar(&window, "w", false, "")

	flag.Var(&dedicated, "dedicated", "Runs as dedicated server, optional number of clients")
	flag.Var(&listen, "listen", "Runs a listen server, optional number of clients")

	flag.IntVar(&bpp, "bpp", -1, "window color depth, negative is unset")
	flag.IntVar(&conSize, "consize", 64, "")
	flag.IntVar(&fsaa, "fsaa", -1, "fsaa level, negative is unset")
	flag.IntVar(&height, "height", -1, "window height, negative is unset")
	flag.IntVar(&particles, "particles", 2048, "")
	flag.IntVar(&port, "port", 26000, "")
	flag.IntVar(&port, "udpport", 26000, "")
	flag.IntVar(&protocol, "protocol", 666, "15: NetQuake, 666: FitzQuake, 999: RMQ") // 666 is FITZQUAKE
	flag.IntVar(&width, "width", -1, "window width, negative is unset")
	flag.IntVar(&zone, "zone", 4*1024*1024, "")

	flag.StringVar(&basedir, "basedir", "", "")
	flag.StringVar(&game, "game", "", "")

	/*
		flag.BoolVar(&isListen, "heapsize", false, "Runs as dedicated server")
		flag.BoolVar(&isListen, "mixspeed", false, "Runs as dedicated server")
		flag.BoolVar(&isListen, "noextmusic", false, "Runs as dedicated server")
		flag.BoolVar(&isListen, "novbo", false, "Runs as dedicated server")
		flag.BoolVar(&isListen, "sndspeed", false, "Runs as dedicated server")
		flag.BoolVar(&isListen, "texturenpot", false, "Runs as dedicated server")
	*/

}

func BaseDirectory() string {
	return basedir
}

func Game() string {
	return game
}

func Height() int {
	return height
}

func Width() int {
	return width
}

func Bpp() int {
	return bpp
}

func Fsaa() int {
	return fsaa
}

func Port() int {
	return port
}

func Particles() int {
	return particles
}

func Protocol() int {
	return protocol
}

func Zone() int {
	return zone
}

func ConsoleSize() int {
	return conSize
}

func ConsoleDebug() bool {
	return conDebug
}

func Current() bool {
	return current
}

func Dedicated() bool {
	return dedicated.set
}

func DedicatedNum() int {
	return dedicated.num
}

func Fitz() bool {
	return fitz
}

func Fullscreen() bool {
	return fullscreen
}

func Hipnotic() bool {
	return hipnotic
}

func Listen() bool {
	return listen.set
}

func ListenNum() int {
	return listen.num
}

func MinMemory() bool {
	return minMemory
}

func Mouse() bool {
	return !noMouse
}

func Sound() bool {
	return !noSound
}

func Quoth() bool {
	return quoth
}

func Rogue() bool {
	return rogue
}

func Window() bool {
	return window
}
