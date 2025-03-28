package printer

import (
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestDebug(t *testing.T) {
	t.Run("debug disable", func(t *testing.T) {
		var printCalled bool

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(fmt.Println, func(_args ...any) (n int, err error) {
			printCalled = true
			return
		})
		patches.ApplyFunc(fmt.Printf, func(_format string, _args ...any) (n int, err error) {
			printCalled = true
			return
		})

		p := NewColorPrinter("test")
		p.Debug("Hello world")
		p.Debugf("Hello %s", "world")

		assert.False(t, printCalled)
	})

	t.Run("debug enable", func(t *testing.T) {
		var printCalled bool

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodReturn(time.Time{}, "Clock", 1, 2, 3)
		patches.ApplyFunc(fmt.Println, func(args ...any) (n int, err error) {
			assert.Len(t, args, 2)
			assert.Equal(t, args[0], "D 01:02 [test]")
			assert.Equal(t, args[1], "Hello world")
			printCalled = true
			return
		})
		patches.ApplyFunc(fmt.Printf, func(format string, args ...any) (n int, err error) {
			assert.Equal(t, format, "D 01:02 [test] Hello %s\n")
			assert.Equal(t, args[0], "world")
			printCalled = true
			return
		})

		p := NewColorPrinter("test")
		p.SetDebug(true)
		p.Debug("Hello world")
		p.Debugf("Hello %s", "world")

		assert.True(t, printCalled)
	})
}

// TODO: ...
