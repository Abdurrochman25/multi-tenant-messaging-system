package config

import (
	"context"
	"database/sql"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type Router struct {
	Routes []fiber.Router
}

type Server struct {
	Config Config
	DB     *sql.DB
	Fiber  *fiber.App
	Router *Router
}

func NewServer(config Config) *Server {
	return &Server{
		Config: config,
		DB:     nil,
		Fiber:  nil,
	}
}

func (s *Server) NewDB(ctx context.Context) error {
	db, err := sql.Open("postgres", s.Config.Database.ConnectionString())
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	s.DB = db
	boil.SetDB(db)
	boil.DebugMode = true

	log.Info("database successfully connected")
	return nil
}
