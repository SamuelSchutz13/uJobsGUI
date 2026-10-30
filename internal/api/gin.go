package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SamuelSchutz13/uJobsGUI/internal/core"
)

func Gin(s *core.Service, static http.Handler) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	reply := func(c *gin.Context, v any, err error) {
		if err != nil {
			c.JSON(Status(err), Body(err))
			return
		}
		c.JSON(http.StatusOK, v)
	}

	g := r.Group("/api")
	g.GET("/info", func(c *gin.Context) { reply(c, Info{s.Root, "gin"}, nil) })
	g.GET("/tracker", func(c *gin.Context) { v, err := s.Tracker(); reply(c, v, err) })
	g.GET("/jobs", func(c *gin.Context) { v, err := s.Jobs(); reply(c, v, err) })
	g.POST("/jobs/status", func(c *gin.Context) {
		var req JobStatusReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, BadRequest(err))
			return
		}
		reply(c, req, s.SetJobStatus(req.Key, req.Status))
	})
	g.GET("/portals", func(c *gin.Context) { v, err := s.Portals(); reply(c, v, err) })
	g.GET("/search", func(c *gin.Context) {
		v, err := s.Search(c.Request.Context(), c.Query("portal"), c.Query("q"))
		if err == nil {
			c.Data(http.StatusOK, "application/json", v)
			return
		}
		reply(c, nil, err)
	})
	g.GET("/commands", func(c *gin.Context) { v, err := s.Commands(); reply(c, v, err) })
	g.GET("/runs", func(c *gin.Context) { reply(c, s.Runs(), nil) })
	g.POST("/runs", func(c *gin.Context) {
		var req StartReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, BadRequest(err))
			return
		}
		v, err := s.Start(req.Command, req.Args)
		reply(c, v, err)
	})
	g.GET("/runs/:id", func(c *gin.Context) { v, err := s.Run(c.Param("id")); reply(c, v, err) })
	g.POST("/runs/:id/reply", func(c *gin.Context) {
		var req ReplyReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, BadRequest(err))
			return
		}
		v, err := s.Reply(c.Param("id"), req.Message)
		reply(c, v, err)
	})
	r.GET("/files/*path", gin.WrapH(s.Files()))
	r.NoRoute(gin.WrapH(static))
	return r
}
