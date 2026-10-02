package cmd

import (
	"fmt"
	"net/http"

	"git.blinkpay.vn/software/authix/config"
	"git.blinkpay.vn/software/authix/graph"
	"git.blinkpay.vn/software/authix/internal/handler"
	"git.blinkpay.vn/software/authix/internal/persistence/postgres"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/blinkpay-software/x/graphqlx"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the Hybrid (GraphQL & REST) API server",
	RunE:  runAPI,
}

func init() {
	rootCmd.AddCommand(apiCmd)
}

func runAPI(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	postgresPersister, err := postgres.NewPersister(
		ctx,
		cfg.Persistence.Postgres.DSN,
	)
	if err != nil {
		return err
	}

	// GraphQL server
	srv := gqlhandler.New(
		graph.NewExecutableSchema(graph.Config{
			Resolvers: &graph.Resolver{
				PostgresPersister: postgresPersister,
				Config:            cfg,
			},
		}),
	)

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.Use(extension.Introspection{})

	srv.AroundOperations(graphqlx.GraphQLAuth)

	// REST server
	restHandler := handler.NewREST(
		postgresPersister,
		&cfg,
	)

	// Use Gorilla Mux for better path params support
	router := mux.NewRouter()

	// GraphQL
	router.Handle("/query", srv)

	// GraphQL Playground
	router.Handle(
		"/",
		playground.Handler(
			"GraphQL playground",
			"/query",
		),
	)
	// REST API - New permission system
	router.HandleFunc("/devices", restHandler.CreateDevice).Methods("POST")
	router.HandleFunc("/devices", restHandler.GetDevices).Methods("GET")
	router.HandleFunc("/devices/{id}", restHandler.GetDeviceByID).Methods("GET")
	router.HandleFunc("/devices/{id}", restHandler.UpdateDevice).Methods("PUT")
	router.HandleFunc("/devices/{id}", restHandler.DeleteDevice).Methods("DELETE")

	router.HandleFunc("/permissions/mask-info", restHandler.GetPermissionMaskInfo).Methods("GET")
	router.HandleFunc("/permissions/assign", restHandler.CreatePermission).Methods("POST")
	router.HandleFunc("/permissions/effective", restHandler.GetEffectivePermissions).Methods("POST")
	router.HandleFunc("/permissions/{id}", restHandler.GetPermission).Methods("GET")
	router.HandleFunc("/permissions/{id}", restHandler.UpdatePermission).Methods("PUT")
	router.HandleFunc("/permissions/{id}", restHandler.DeletePermission).Methods("DELETE")

	addr := cfg.Server.HTTPAddress

	log.Info().
		Str("addr", addr).
		Msg("starting Hybrid (GraphQL & REST) server")

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal().Err(err).Str("addr", addr).Msg("HTTP server failed")
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}
