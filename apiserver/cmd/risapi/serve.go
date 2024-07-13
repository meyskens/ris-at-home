package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"net/http"
	"os"
	"os/signal"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/meyskens/ris-at-home/apiserver/pkg/ris"
	"github.com/meyskens/ris-at-home/apiserver/pkg/ris/delijn"
	"github.com/meyskens/ris-at-home/apiserver/pkg/ris/irail"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(NewServeCmd())
}

type serveCmdOptions struct {
	BindAddr string
	Port     int
}

// NewServeCmd generates the `serve` command
func NewServeCmd() *cobra.Command {
	s := serveCmdOptions{}
	c := &cobra.Command{
		Use:     "serve",
		Short:   "Serves the HTTP REST endpoint",
		Long:    `Serves the HTTP REST endpoint on the given bind address and port`,
		PreRunE: s.Validate,
		RunE:    s.RunE,
	}
	c.Flags().StringVarP(&s.BindAddr, "bind-address", "b", "0.0.0.0", "address to bind port to")
	c.Flags().IntVarP(&s.Port, "port", "p", 8080, "Port to listen on")

	return c
}

func (s *serveCmdOptions) Validate(cmd *cobra.Command, args []string) error {
	return nil
}

func (s *serveCmdOptions) RunE(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// serve static files from public directory
	e.Static("/", "public")

	// handle API calls
	e.GET("/db/apis/ris-boards/v1/public/departures/:id", func(c echo.Context) error {
		stations := strings.Split(c.Param("id"), ",")
		if stations[0] == "" {
			stations = []string{"008821006"}
		}
		resp := ris.DeparturesResponse{
			Departures:  []ris.Departure{},
			Disruptions: []any{},
		}

		for _, station := range stations {
			var err error
			var departures []ris.Departure
			if strings.HasPrefix(station, "008") {
				departures, err = irail.LiveboardToRISDepartures(station, "nl")
			} else {
				departures, err = delijn.LiveboardToRISDepartures(station)
			}
			if err != nil {
				return c.JSON(http.StatusInternalServerError, err)
			}

			resp.Departures = append(resp.Departures, departures...)
		}

		// sort resp.Departures on TimeSchedule
		sort.Slice(resp.Departures, func(i, j int) bool {
			return resp.Departures[i].TimeSchedule.Before(resp.Departures[j].TimeSchedule)
		})

		return c.JSON(http.StatusOK, resp)
	})

	go func() {
		e.Start(fmt.Sprintf("%s:%d", s.BindAddr, s.Port))
		cancel() // server ended, stop the world
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			return nil
		}
	}
}
