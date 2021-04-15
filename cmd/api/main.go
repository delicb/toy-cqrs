package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/multierr"
	"golang.org/x/crypto/bcrypt"

	"github.com/delicb/toy-cqrs/types"
)

func main() {
	rootContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := NewDBManager(rootContext, os.Getenv("DATABASE_URL"))
	nats := NewNatsManager(os.Getenv("NATS_URL"))
	httpServer := &server{
		nats: nats,
		db:   db,
	}
	app := echo.New()
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())

	app.GET("/:id", httpServer.getUser)
	app.POST("/register", httpServer.registerUser)
	app.POST("/emailChange", httpServer.emailChange)
	app.PUT("/enableUser/:id", httpServer.enableUser)
	app.Logger.Fatal(app.Start("0.0.0.0:8001"))
}

type server struct {
	nats NatsManager
	db   DBManager
}

func (s *server) getUser(c echo.Context) error {
	user, err := s.db.GetUser(c.Param("id"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *server) registerUser(c echo.Context) error {
	c.Logger().Info("registering new user")
	request := &userCreateRequest{}
	if err := (&echo.DefaultBinder{}).BindBody(c, request); err != nil {
		c.Logger().Error("failed to bind body to the request: %v", err)
		return err
	}

	if err := request.Validate(); err != nil {
		return err
	}

	hashedPwd, err := hashPassword(request.Password)
	if err != nil {
		return err
	}
	resp, err := s.executeCommandSync(types.CreateUserCmdID, &types.CreateUserCmdParams{
		Email:    request.Email,
		Password: hashedPwd,
	})
	if err != nil {
		return err
	}

	user, err := s.db.GetUser(resp)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, user)
}

func (s *server) emailChange(c echo.Context) error {
	c.Logger().Debug("changing user email")
	request := &emailChangeRequest{}
	if err := (&echo.DefaultBinder{}).BindBody(c, request); err != nil {
		c.Logger().Errorf("failed to bind body to the request: %v", err)
		return err
	}

	if err := request.Validate(); err != nil {
		return err
	}

	resp, err := s.executeCommandSync(types.ModifyUserCmdID, &types.ModifyUserCmdParams{
		ID:    request.ID,
		Email: &request.Email,
	})
	if err != nil {
		c.Logger().Error("got error during command execution: %v", err)
		return err
	}

	if resp != request.ID {
		c.Logger().Error("got wrong user in response")
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	user, err := s.db.GetUser(resp)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *server) enableUser(c echo.Context) error {
	c.Logger().Debug("enabling user")
	userID := c.Param("id")
	enabled := true
	resp, err := s.executeCommandSync(types.ModifyUserCmdID, &types.ModifyUserCmdParams{
		ID:        userID,
		IsEnabled: &enabled,
	})
	if err != nil {
		c.Logger().Error("got error during command execution: %v", err)
		return err
	}
	if resp != userID {
		c.Logger().Error("got wrong user in response")
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	user, err := s.db.GetUser(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *server) executeCommandSync(cmdID types.CommandID, params interface{}) (resp string, err error) {
	cmd, err := types.NewCommand(cmdID, params)
	if err != nil {
		return "", fmt.Errorf("failed to create a command: %w", err)
	}

	// start listening for response before command is actually sent
	_, _, err = s.nats.Subscribe(cmd.CorrelationID())
	if err != nil {
		return "", err
	}
	defer func() {
		unsubErr := s.nats.Unsubscribe(cmd.CorrelationID())
		if unsubErr != nil {
			err = multierr.Combine(err, unsubErr)
		}
	}()

	// send command
	if err := s.nats.SendCommand(cmd); err != nil {
		return "", err
	}

	// wait for the effect of executing command to reach us
	return s.nats.WaitForEvent(cmd.CorrelationID(), 5*time.Second)
}

func hashPassword(in string) (string, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(in), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("bcrypt:%s", string(pwd)), nil
}
