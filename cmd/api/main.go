package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nats-io/nats.go"
	"golang.org/x/crypto/bcrypt"

	"github.com/delicb/toy-cqrs/users"
)

func main() {
	rootContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := NewDBManager(rootContext, os.Getenv("DATABASE_URL"))
	natsConn, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(err)
	}
	usersClient := users.NewClient(natsConn)

	httpServer := &server{
		db:    db,
		users: usersClient,
	}
	app := echo.New()
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())

	app.GET("/users/:id", httpServer.getUser)
	app.POST("/users/register", httpServer.registerUser)
	app.PUT("/users/:id/emailChange", httpServer.emailChange)
	app.PUT("/users/:id/passwordChange", httpServer.passwordChange)
	app.PUT("/users/:id/enable", httpServer.enableUser)
	app.PUT("/users/:id/disable", httpServer.disableUser)
	app.Logger.Fatal(app.Start("0.0.0.0:8001"))
}

type server struct {
	db    DBManager
	users users.Client
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
	userID, err := s.users.Create(request.Email, hashedPwd)
	if err != nil {
		return err
	}

	user, err := s.db.GetUser(userID)
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

	userID := c.Param("id")
	err := s.users.ChangeEmail(userID, request.Email)
	if err != nil {
		c.Logger().Errorf("got error during command execution: %v", err)
		return err
	}

	user, err := s.db.GetUser(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *server) passwordChange(c echo.Context) error {
	c.Logger().Debug("changing user password")
	request := &passwordChangeRequest{}
	if err := (&echo.DefaultBinder{}).BindBody(c, request); err != nil {
		c.Logger().Errorf("failed to bind body to the request: %v", err)
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	userID := c.Param("id")
	hashedPwd, err := hashPassword(request.Password)
	if err != nil {
		return err
	}
	err = s.users.ChangePassword(userID, hashedPwd)
	if err != nil {
		c.Logger().Errorf("got error during command execution: %v", err)
		return err
	}
	user, err := s.db.GetUser(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *server) enableUser(c echo.Context) error {
	c.Logger().Debug("enabling user")
	userID := c.Param("id")
	if err := s.users.Enable(userID); err != nil {
		c.Logger().Errorf("got error during command execution: %v", err)
		return err
	}

	user, err := s.db.GetUser(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *server) disableUser(c echo.Context) error {
	c.Logger().Debug("disabling user")
	userID := c.Param("id")
	if err := s.users.Disable(userID); err != nil {
		c.Logger().Errorf("got error during command execution: %v", err)
		return err
	}

	user, err := s.db.GetUser(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func hashPassword(in string) (string, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(in), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("bcrypt:%s", string(pwd)), nil
}
