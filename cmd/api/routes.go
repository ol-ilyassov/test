package main

import (
	"expvar"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/", app.home)

	router.HandlerFunc(http.MethodGet, "/user", app.user)

	//router.HandlerFunc(http.MethodGet, "/healthcheck", app.healthcheckHandler)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	//return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	//router.Handle(http.MethodGet,"/static/", http.StripPrefix("/static", fileServer))
	router.Handler(http.MethodGet, "/static/", http.StripPrefix("/static", fileServer))

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(router))))
}
