package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	// HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful Shutdown
	shutdownError := make(chan error) // Errors from Graceful Shutdown

	go func() {
		// Quit channel with os.Signal values.
		quit := make(chan os.Signal, 1)
		// Listen for incoming SIGINT and SIGTERM signals.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		// nil = success, or error (, or 5-second context deadline)
		app.wg.Wait()
		shutdownError <- nil
	}()

	// Log "Starting server" message.
	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// wait for error after Graceful Shutdown (If exists).
	err = <-shutdownError
	if err != nil {
		return err
	}

	// Successful Graceful Shutdown
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}

// SIGINT - signal: interrupt - [CTRL+C]
// SIGTERM - signal: terminated
// SIGKILL and SIGQUIT - no caught signal (killed).
