package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/benbjohnson/clock"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppRender(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	scr := NewTestScreen(t, 80, 40)
	clock := clock.NewMock()
	widget := NewMockWidget(ctrl)

	app := App{
		Root:   widget,
		Screen: scr,
		Clock:  clock,
		Log:    logtest.NewLogger(t),
		FPS:    1, // keep the math for time below simple
	}
	app.Start()
	defer func() {
		app.Stop()
		assert.NoError(t, app.Wait())
	}()

	// There's a small race condition here, and since we don't have any
	// hook into things actually getting drawn onto the screen, make it a
	// bit fuzzy: leave some slack.
	widget.EXPECT().Draw(gomock.Any()).MinTimes(90).MaxTimes(100)
	for i := 0; i < 100; i++ {
		clock.Add(time.Second)
	}
}

func TestAppEvents(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	scr := NewTestScreen(t, 80, 40)
	clock := clock.NewMock()

	widget := NewMockWidget(ctrl)
	widget.EXPECT().Draw(gomock.Any()).AnyTimes()

	app := App{
		Root:   widget,
		Screen: scr,
		Clock:  clock,
		Log:    logtest.NewLogger(t),
	}
	app.Start()
	defer func() {
		app.Stop()
		assert.NoError(t, app.Wait())
	}()

	t.Run("resize", func(t *testing.T) {
		scr.SetSize(100, 60)
	})

	t.Run("handled action", func(t *testing.T) {
		widget.EXPECT().
			HandleEvent(gomock.Any()).
			Return(true)

		scr.InjectKey(tcell.KeyRune, 'f', 0)
	})

	t.Run("quit", func(t *testing.T) {
		scr.InjectKey(tcell.KeyEscape, 0, 0)

		// If this deadlocks, esc didn't quit.
		assert.NoError(t, app.Wait())
	})
}

func TestAppPanic(t *testing.T) {
	t.Parallel()

	assertPanic := func(t *testing.T, app *App, buff *bytes.Buffer) {
		t.Helper()

		err := app.Wait()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "great sadness")
		assert.Contains(t, buff.String(), "panic: great sadness")
		assert.Contains(t, buff.String(), "TestAppPanic")
		assert.Contains(t, buff.String(), "app_test.go")
	}

	t.Run("event panic", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		scr := NewTestScreen(t, 80, 40)

		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).AnyTimes()

		var buff bytes.Buffer
		app := App{
			Root:   widget,
			Screen: scr,
			Log:    log.New(&buff),
		}
		app.Start()

		widget.EXPECT().
			HandleEvent(gomock.Any()).
			Do(func(tcell.Event) {
				panic("great sadness")
			})

		scr.InjectKey(tcell.KeyRune, 'f', 0)
		assertPanic(t, &app, &buff)
	})

	t.Run("render panic", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		scr := NewTestScreen(t, 80, 40)

		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).
			Do(func(tcell.Screen) {
				panic("great sadness")
			})

		var buff bytes.Buffer
		app := App{
			Root:   widget,
			Screen: scr,
			Log:    log.New(&buff),
		}
		app.Start()

		assertPanic(t, &app, &buff)
	})
}
