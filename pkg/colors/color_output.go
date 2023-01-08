package colors

import "runtime"

type colors struct {
	Cyan   string
	Yellow string
	Red    string
	Reset  string
}

func SetColor() *colors {
	c := &colors{}

	if runtime.GOOS == "windows" {
		c.Cyan = ""
		c.Yellow = ""
		c.Red = ""
		c.Reset = ""
	} else {
		c.Cyan = "\033[36m"
		c.Yellow = "\033[33m"
		c.Red = "\033[1;31m"
		c.Reset = "\033[0m"
	}

	return c

}
