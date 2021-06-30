package main

import (
	"expvar"
	"fmt"
	"github.com/felixge/httpsnoop"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// Builtin recover function to check if there has been a panic or not.
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				// ERROR level and send the client a 500 Internal Server Error response.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// Hold the clients' IP addresses and rate limiters.
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			// Loop through all clients. If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	// This is a closure, which 'closes over' the limiter variable.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if app.config.limiter.enabled {
			// Extract the client's IP address from the request.
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			mu.Lock()
			// If IP address dont exist in map, then initialize a new rate limiter
			//and add the IP address and limiter to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}

			clients[ip].lastSeen = time.Now()

			// Call the Allow() method on the rate limiter for the current IP address.
			// If not allowed, then 429 Too Many Requests response.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

// Auth User with token
//func (app *application) authenticate(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Add("Vary", "Authorization")
//		// Retrieve the value of the Authorization header from the request.
//		authorizationHeader := r.Header.Get("Authorization")
//		// Add the AnonymousUser to the request context.
//		if authorizationHeader == "" {
//			r = app.contextSetUser(r, data.AnonymousUser)
//			next.ServeHTTP(w, r)
//			return
//		}
//
//		headerParts := strings.Split(authorizationHeader, " ")
//		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
//			app.invalidAuthenticationTokenResponse(w, r)
//			return
//		}
//
//		token := headerParts[1]
//
//		v := validator.New()
//		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
//			app.invalidAuthenticationTokenResponse(w, r)
//			return
//		}
//		// Retrieve the details of the user associated with the authentication token.
//		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
//		if err != nil {
//			switch {
//			case errors.Is(err, data.ErrRecordNotFound):
//				app.invalidAuthenticationTokenResponse(w, r)
//			default:
//				app.serverErrorResponse(w, r, err)
//			}
//			return
//		}
//		// Add the user information to the request context.
//		r = app.contextSetUser(r, user)
//
//		next.ServeHTTP(w, r)
//	})
//}
//
//func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		user := app.contextGetUser(r)
//		// Is User Anonymous?
//		if user.IsAnonymous() {
//			app.authenticationRequiredResponse(w, r)
//			return
//		}
//		next.ServeHTTP(w, r)
//	})
//}
//
//// Checks that a user is both authenticated and activated.
//func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
//	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		user := app.contextGetUser(r)
//		// Check that a user is activated.
//		if !user.Activated {
//			app.inactiveAccountResponse(w, r)
//			return
//		}
//		next.ServeHTTP(w, r)
//	})
//	// Wrap with requireAuthenticatedUser().
//	return app.requireAuthenticatedUser(fn)
//}
//
//func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
//	fn := func(w http.ResponseWriter, r *http.Request) {
//		user := app.contextGetUser(r)
//		// Get the slice of permissions for the user.
//		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
//		if err != nil {
//			app.serverErrorResponse(w, r, err)
//			return
//		}
//		if !permissions.Include(code) {
//			app.notPermittedResponse(w, r) // 403 Forbidden response.
//			return
//		}
//		// Next handler in the chain.
//		next.ServeHTTP(w, r)
//	}
//	// Wrap with requireActivatedUser().
//	return app.requireActivatedUser(fn)
//}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")
		if origin != "" && len(app.config.cors.trustedOrigins) != 0 {
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					// If request has the HTTP method OPTIONS and "Access-Control-Request-Method" header,
					// then it as a preflight request.
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						// Set the necessary preflight response headers
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) metrics(next http.Handler) http.Handler {
	// Init expvar variables when the middleware chain is first built.
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")
	// Declare a new expvar map to hold the count of responses for each HTTP status code.
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequestsReceived.Add(1)
		// Returns the metrics struct of handlers.
		metrics := httpsnoop.CaptureMetrics(next, w, r)

		totalResponsesSent.Add(1)
		// Get the request processing time in microseconds
		// Increment the cumulative processing time.
		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())
		// Increment the count for specific status code by 1.
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	})
}
