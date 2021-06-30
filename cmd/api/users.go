package main

import (
	"github.com/ol-ilyassov/test/internal/data"
	"net/http"
)

func (app *application) user(w http.ResponseWriter, r *http.Request) {

	user := &data.User{
		Name:    "Alibek",
		Email:   "a.sovetkazhiyev@gmail.com",
		Version: 1,
	}

	app.render(w, r, "show.page.tmpl", &templateData{
		User: user,
		//UserID:  app.session.GetInt(r, "authenticatedUserID"),
	})
}
