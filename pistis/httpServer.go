package pistis

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/cors"
	"github.com/martini-contrib/render"
)

var(
	httpServerURL string
)

func StartHttpServer(frontendServer string){

	m := martini.Classic()

	/*m.Use(csrf.Generate(&csrf.Options{
		Secret:     "token123",
		SessionKey: "userID",
		//
		ErrorFunc: func(w http.ResponseWriter) {
			http.Error(w, "CSRF token validation failed", http.StatusBadRequest)
		},
	}))*/

	m.Use(cors.Allow(&cors.Options{
		AllowOrigins:     []string{frontendServer},
		AllowMethods:     []string{"POST", "GET", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type","Content-Range","Content-Disposition", "Accept","Authorization","Set-Cookie",},
		ExposeHeaders:    []string{"Authorization","Set-Cookie",},
		AllowCredentials: true,
	}))

	m.Use(Authenticator)
	//m.Use(pistis.CDNServer)
	m.Use(render.Renderer(render.Options{
		// nothing here to configure
	}))

	r := martini.NewRouter()


	AuthServer(r)

	CDNServer(r)

	m.MapTo(r, (*martini.Routes)(nil))

	m.Action(r.Handle)

	m.Run()
}