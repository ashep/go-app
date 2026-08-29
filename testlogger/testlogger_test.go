package testlogger_test

import (
	"testing"

	"github.com/ashep/go-app/testlogger"
	"github.com/stretchr/testify/assert"
)

func TestLogger(main *testing.T) {
	main.Parallel()

	main.Run("Content", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message")

		assert.Equal(t, `{"level":"info","message":"test message"}
`, tl.Content())
	})

	main.Run("Messages", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message 1")
		l.Warn().Msg("test message 2")
		l.Error().Msg("test message 3")

		assert.Equal(t, []string{
			`{"level":"info","message":"test message 1"}`,
			`{"level":"warn","message":"test message 2"}`,
			`{"level":"error","message":"test message 3"}`,
		}, tl.Messages())
	})

	main.Run("MessagesContainsAny", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message 1")
		l.Warn().Msg("test message 2")
		l.Error().Msg("test message 3")

		assert.Equal(t, []string{
			`{"level":"warn","message":"test message 2"}`,
			`{"level":"error","message":"test message 3"}`,
		}, tl.MessagesContainsAny("test message 2", "test message 3"))
	})

	main.Run("FirstMessageContains", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message 1")
		l.Warn().Msg("test message 2")
		l.Error().Msg("test message 3")

		assert.Equal(t, []string{
			`{"level":"warn","message":"test message 2"}`,
		}, tl.MessagesContainsAny("2"))
	})

	main.Run("MessagesContainsAll", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message 1")
		l.Warn().Msg("test message 2")
		l.Error().Msg("test message 3")

		assert.Equal(t, []string{
			`{"level":"warn","message":"test message 2"}`,
		}, tl.MessagesContainsAll("warn", "message 2"))

		assert.Empty(t, tl.MessagesContainsAll("warn", "message 3"))
	})

	main.Run("FirstMessageContainsAny", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message 1")
		l.Warn().Msg("test message 2")

		assert.Equal(t, `{"level":"warn","message":"test message 2"}`, tl.FirstMessageContainsAny("message 2"))
		assert.Equal(t, "", tl.FirstMessageContainsAny("nope"))
	})

	main.Run("FirstMessageContainsAll", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message 1")
		l.Warn().Msg("test message 2")

		assert.Equal(t, `{"level":"warn","message":"test message 2"}`, tl.FirstMessageContainsAll("warn", "message 2"))
		assert.Equal(t, "", tl.FirstMessageContainsAll("warn", "message 1"))
	})

	main.Run("AssertContains", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message")

		tl.AssertContains("test message")
	})

	main.Run("AssertNotContains", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message")

		tl.AssertNotContains("other message")
	})

	main.Run("AssertNoErrors", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message")
		l.Warn().Msg("test warn")

		tl.AssertNoErrors()
	})

	main.Run("AssertContainsError", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Error().Msg("test error")

		tl.AssertContainsError()
	})

	main.Run("AssertHasError", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("unrelated")
		l.Error().Msg("something failed")

		tl.AssertHasError("something failed")
	})

	main.Run("AssertNoWarns", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message")
		l.Error().Msg("test error")

		tl.AssertNoWarns()
	})

	main.Run("AssertContainsWarn", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Warn().Msg("test warn")

		tl.AssertContainsWarn()
	})

	main.Run("AssertHasWarn", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("unrelated")
		l.Warn().Msg("something risky")

		tl.AssertHasWarn("something risky")
	})

	main.Run("AssertNoWarnsAndErrors", func(t *testing.T) {
		t.Parallel()

		tl := testlogger.New(t)
		l := tl.Logger()

		l.Info().Msg("test message")

		tl.AssertNoWarnsAndErrors()
	})
}
