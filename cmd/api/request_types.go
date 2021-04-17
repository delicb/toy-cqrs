package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type userCreateRequest struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

func (r *userCreateRequest) Validate() error {
	if r.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}
	if r.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "password is required")
	}
	if len(r.Password) < 3 { // very lax security
		return echo.NewHTTPError(http.StatusBadRequest, "password is too short")
	}
	return nil
}

type emailChangeRequest struct {
	Email string `json:"email"`
}

func (e *emailChangeRequest) Validate() error {
	// basic validation
	if e.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}
	return nil
}

type passwordChangeRequest struct {
	Password string `json:"password,omitempty"`
}

func (p *passwordChangeRequest) Validate() error {
	if p.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "password is required")
	}
	if len(p.Password) < 3 {
		return echo.NewHTTPError(http.StatusBadRequest, "password is too short")
	}
	return nil
}
