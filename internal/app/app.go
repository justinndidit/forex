package app

import (
	"github.com/justinndidit/forex/internal/config"
	"github.com/justinndidit/forex/internal/database"
	"github.com/justinndidit/forex/internal/handler"
	"github.com/justinndidit/forex/internal/repository"
	"github.com/justinndidit/forex/internal/util"
	"github.com/rs/zerolog"
)

type Application struct {
	Config  *config.Config
	Logger  *zerolog.Logger
	DB      *database.Database
	Handler *handler.ForexHandler
	repo    *repository.ForexRepository
	ImgGen  *util.ImageService
}

func NewApp(cfg *config.Config, logger *zerolog.Logger, db *database.Database) *Application {
	repo := repository.NewForexRepository(logger, db)
	imgGen := util.NewImageService(logger)

	handler := handler.NewForexHandler(logger, db, repo, imgGen)
	return &Application{
		Config:  cfg,
		Logger:  logger,
		DB:      db,
		repo:    repo,
		Handler: handler,
		ImgGen:  imgGen,
	}
}
