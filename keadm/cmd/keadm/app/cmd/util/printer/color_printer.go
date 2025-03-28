package printer

import (
	"bufio"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/fatih/color"
)

type ColorPrinter struct {
	tag   string
	debug bool
	step  int
}

// Check ColorPrinter implements Printer interface.
var _ Printer = (*ColorPrinter)(nil)

func NewColorPrinter(tag string) *ColorPrinter {
	return &ColorPrinter{
		tag: tag,
	}
}

func (c *ColorPrinter) SetDebug(debug bool) {
	c.debug = debug
}

func (c *ColorPrinter) Debug(msg string) {
	if !c.debug {
		return
	}
	fmt.Println(c.header("D"), msg)
}

func (c *ColorPrinter) Debugf(format string, args ...any) {
	if !c.debug {
		return
	}
	fmt.Printf(c.header("D")+" "+format+"\n", args...)
}

func (c *ColorPrinter) Info(msg string) {
	fmt.Println(c.header("I"), msg)
}

func (c *ColorPrinter) Infof(format string, args ...any) {
	fmt.Printf(c.header("I")+" "+format+"\n", args...)
}

func (c *ColorPrinter) Warn(msg string) {
	color.Yellow(fmt.Sprintln(c.header("W"), msg))
}

func (c *ColorPrinter) Warnf(format string, args ...any) {
	color.Yellow(c.header("W")+" "+format+"\n", args...)
}

func (c *ColorPrinter) Error(err error, msg string) {
	color.Red(fmt.Sprintf("%s %s, err: %v\n", c.header("E"), msg, err))
	if c.debug {
		debug.PrintStack()
	}
}

func (c *ColorPrinter) Errorf(err error, format string, args ...any) {
	color.Red(c.header("E")+" "+format+"\n", args...)
	if c.debug {
		debug.PrintStack()
	}
}

func (c *ColorPrinter) Step(msg string) StepDone {
	c.step++
	fmt.Println(fmt.Sprintf("%s %d.", c.header("I"), c.step), msg)
	step := c.step // Make the callback function print the value of this step only.
	return func() {
		c.stepDone(step)
	}
}

func (c *ColorPrinter) Stepf(format string, args ...any) StepDone {
	c.step++
	args = append([]any{c.step}, args...)
	fmt.Printf(c.header("I")+" %d. "+format+"\n", args...)
	step := c.step // Make the callback function print the value of this step only.
	return func() {
		c.stepDone(step)
	}
}

func (c *ColorPrinter) stepDone(step int) {
	color.Green("%s Step %d [Done].\n", c.header("I"), step)
}

func (c *ColorPrinter) StepAndRun(fn func() error, msg string) error {
	c.step++
	fmt.Printf("%s %d. %s\n", c.header("I"), c.step, msg)
	if err := fn(); err != nil {
		return err
	}
	c.stepDone(c.step)
	return nil
}

func (c *ColorPrinter) StepfAndRun(fn func() error, format string, args ...any) error {
	c.step++
	args = append([]any{c.step}, args...)
	fmt.Printf(c.header("I")+" %d. "+format, args...)
	if err := fn(); err != nil {
		return err
	}
	c.stepDone(c.step)
	return nil
}

func (c *ColorPrinter) Input(msg string) (string, error) {
	fmt.Print(c.header("I"), " ", msg)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return scanner.Text(), nil
}

func (c *ColorPrinter) Inputf(format string, args ...any) (string, error) {
	fmt.Printf(c.header("I")+" "+format, args...)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return scanner.Text(), nil
}

func (c *ColorPrinter) header(logType string) string {
	now := time.Now()
	h, m, _ := now.Clock()
	res := fmt.Sprintf("%s %02d:%02d", logType, h, m)
	if c.tag != "" {
		res = res + " [" + c.tag + "]"
	}
	return res
}
