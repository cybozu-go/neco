package placemat

import (
	"fmt"
	"hash/fnv"
	"io"
	"strings"

	"github.com/fatih/color"
)

var colors = []color.Attribute{
	color.FgRed,
	color.FgGreen,
	color.FgYellow,
	color.FgBlue,
	color.FgMagenta,
	color.FgCyan,
}

func chooseColor(data string) color.Attribute {
	h := fnv.New32a()
	_, err := h.Write([]byte(data))
	if err != nil {
		panic(err)
	}
	num := int(h.Sum32() % uint32(len(colors)))
	return colors[num]
}

func newColoredLogWriter(kind, name string, writer io.Writer) *coloredLogWriter {
	c := chooseColor(name)
	return &coloredLogWriter{
		prefix: fmt.Sprintf("[%s] %s - ", kind, name),
		writer: writer,
		color:  color.New(c),
	}
}

type coloredLogWriter struct {
	prefix string
	writer io.Writer
	color  *color.Color
	prev   string
}

func (w *coloredLogWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	input := string(p)

	input = strings.Replace(input, "\r\n", "\n", -1)
	input = w.prev + input

	inputs := strings.Split(input, "\n")
	w.prev = inputs[len(inputs)-1]
	inputs = inputs[:len(inputs)-1]
	for _, s := range inputs {
		_, err = w.color.Fprintln(w.writer, w.prefix+s)
		if err != nil {
			return
		}
	}

	return
}
