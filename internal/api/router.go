package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
)

type router struct {
	handler           handler.Handler
	corsAllowedOrigin string
}

func NewRouter(handler handler.Handler, corsAllowedOrigin string) *router {
	return &router{
		handler:           handler,
		corsAllowedOrigin: corsAllowedOrigin,
	}
}

func (rt *router) RegisterHandlers() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{rt.corsAllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// if app.config.rateLimiter.Enabled {
	// 	r.Use(app.RateLimiterMiddleware)
	// }
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		// Operations
		// r.Get("/health", app.healthCheckHandler)
		// r.With(app.BasicAuthMiddleware()).Get("/debug/vars", expvar.Handler().ServeHTTP)

		// docsURL := fmt.Sprintf("%s/swagger/doc.json", app.config.addr)
		// r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL(docsURL)))

		r.Route("/posts", func(r chi.Router) {
			// r.Use(app.AuthTokenMiddleware)
			r.Post("/", rt.handler.PostHandler.CreatePostHandler)

			r.Route("/{postID}", func(r chi.Router) {
				//r.Use(app.postsContextMiddleware)
				r.Get("/", rt.handler.PostHandler.GetPostHandler)
				r.Put("/", rt.handler.PostHandler.UpdatePostHandler)
				r.Delete("/", rt.handler.PostHandler.DeletePostHandler)
			})
		})

		r.Route("/search/posts", func(r chi.Router) {
			// r.Use(app.AuthTokenMiddleware)
			r.Post("/", rt.handler.PostHandler.SearchPostHandler)
		})

		r.Route("/users", func(r chi.Router) {
			//r.Put("/activate/{token}", app.activateUserHandler)

			r.Route("/{userID}", func(r chi.Router) {
				//r.Use(app.AuthTokenMiddleware)

				r.Get("/", rt.handler.UserHandler.GetUserHandler)
				r.Put("/", rt.handler.UserHandler.UpdateUserHandler)
				//r.Put("/follow", app.followUserHandler)
				//r.Put("/unfollow", app.unfollowUserHandler)
			})

			// r.Group(func(r chi.Router) {
			// 	r.Use(app.AuthTokenMiddleware)
			// 	r.Get("/feed", app.getUserFeedHandler)
			// })
		})

		// Public routes
		r.Route("/authentication", func(r chi.Router) {
			r.Post("/user", rt.handler.UserHandler.RegisterUserHandler)
			//r.Post("/token", app.createTokenHandler)
		})
	})

	return r
}
