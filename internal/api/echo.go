package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/SamuelSchutz13/uJobsGUI/internal/core"
)

func Echo(s *core.Service, static http.Handler) http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())

	reply := func(c echo.Context, v any, err error) error {
		if err != nil {
			return c.JSON(Status(err), Body(err))
		}
		return c.JSON(http.StatusOK, v)
	}

	g := e.Group("/api")
	g.GET("/info", func(c echo.Context) error { return reply(c, Info{s.Root, "echo"}, nil) })
	g.GET("/tracker", func(c echo.Context) error { v, err := s.Tracker(); return reply(c, v, err) })
	g.GET("/jobs", func(c echo.Context) error { v, err := s.Jobs(); return reply(c, v, err) })
	g.POST("/jobs/status", func(c echo.Context) error {
		var req JobStatusReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, BadRequest(err))
		}
		return reply(c, req, s.SetJobStatus(req.Key, req.Status))
	})
	g.GET("/portals", func(c echo.Context) error { v, err := s.Portals(); return reply(c, v, err) })
	g.GET("/search", func(c echo.Context) error {
		v, err := s.Search(c.Request().Context(), c.QueryParam("portal"), c.QueryParam("q"))
		if err != nil {
			return reply(c, nil, err)
		}
		return c.JSONBlob(http.StatusOK, v)
	})
	g.GET("/commands", func(c echo.Context) error { v, err := s.Commands(); return reply(c, v, err) })
	g.GET("/runs", func(c echo.Context) error { return reply(c, s.Runs(), nil) })
	g.POST("/runs", func(c echo.Context) error {
		var req StartReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, BadRequest(err))
		}
		v, err := s.Start(req.Command, req.Args)
		return reply(c, v, err)
	})
	g.GET("/runs/:id", func(c echo.Context) error { v, err := s.Run(c.Param("id")); return reply(c, v, err) })
	g.POST("/runs/:id/reply", func(c echo.Context) error {
		var req ReplyReq
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, BadRequest(err))
		}
		v, err := s.Reply(c.Param("id"), req.Message)
		return reply(c, v, err)
	})
	e.GET("/files/*", echo.WrapHandler(s.Files()))
	e.GET("/*", echo.WrapHandler(static))
	return e
}
