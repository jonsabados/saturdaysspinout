package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/jonsabados/saturdaysspinout/correlation"
)

type ErrorResponse struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func DoErrorResponse(ctx context.Context, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusInternalServerError)
	bytes, err := json.Marshal(ErrorResponse{
		Message:       "An unexpected error has been encountered. Please reference the included correlation id in any support inquires.",
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling ErrorResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type RequestErrors struct {
	Errors        []string     `json:"errors"`
	FieldErrors   []FieldError `json:"fieldErrors"`
	CorrelationID string       `json:"correlationId"`
}

func NewRequestErrors() RequestErrors {
	return RequestErrors{
		Errors:      []string{},
		FieldErrors: []FieldError{},
	}
}

func (r RequestErrors) WithError(error string) RequestErrors {
	return RequestErrors{
		Errors:        append(slices.Clone(r.Errors), error),
		FieldErrors:   r.FieldErrors,
		CorrelationID: r.CorrelationID,
	}
}

func (r RequestErrors) WithFieldError(field, error string) RequestErrors {
	return RequestErrors{
		Errors: r.Errors,
		FieldErrors: append(slices.Clone(r.FieldErrors), FieldError{
			Field: field,
			Error: error,
		}),
		CorrelationID: r.CorrelationID,
	}
}

func (r RequestErrors) HasAnyError() bool {
	return len(r.Errors) > 0 || len(r.FieldErrors) > 0
}

func DoBadRequestResponse(ctx context.Context, result RequestErrors, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusBadRequest)
	result.CorrelationID = correlation.FromContext(ctx)
	bytes, err := json.Marshal(result)
	if err != nil {
		panic(fmt.Errorf("error marshalling BadRequestResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type AcceptedResponse struct {
	Response      interface{} `json:"response"`
	CorrelationID string      `json:"correlationId"`
}

func DoAcceptedResponse(ctx context.Context, Response interface{}, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusAccepted)
	bytes, err := json.Marshal(AcceptedResponse{
		Response:      Response,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling AcceptedResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type UnauthorizedResponse struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func DoUnauthorizedResponse(ctx context.Context, message string, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusUnauthorized)
	bytes, err := json.Marshal(UnauthorizedResponse{
		Message:       message,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling UnauthorizedResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type ForbiddenResponse struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func DoForbiddenResponse(ctx context.Context, message string, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusForbidden)
	bytes, err := json.Marshal(ForbiddenResponse{
		Message:       message,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling ForbiddenResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type NotFoundResponse struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func DoNotFoundResponse(ctx context.Context, message string, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusNotFound)
	bytes, err := json.Marshal(NotFoundResponse{
		Message:       message,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling NotFoundResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type OKResponse struct {
	Response      interface{} `json:"response"`
	CorrelationID string      `json:"correlationId"`
}

func DoOKResponse(ctx context.Context, Response interface{}, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusOK)
	bytes, err := json.Marshal(OKResponse{
		Response:      Response,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling OKResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type Pagination struct {
	Page           int `json:"page"`
	ResultsPerPage int `json:"resultsPerPage"`
	TotalResults   int `json:"totalResults"`
	TotalPages     int `json:"totalPages"`
}

type OKListResponse[T any] struct {
	Items         []T        `json:"items"`
	Pagination    Pagination `json:"pagination"`
	CorrelationID string     `json:"correlationId"`
}

func DoOKListResponse[T any](ctx context.Context, items []T, page, resultsPerPage, totalResults int, writer http.ResponseWriter) {
	totalPages := totalResults / resultsPerPage
	if totalResults%resultsPerPage != 0 {
		totalPages++
	}

	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusOK)
	bytes, err := json.Marshal(OKListResponse[T]{
		Items: items,
		Pagination: Pagination{
			Page:           page,
			ResultsPerPage: resultsPerPage,
			TotalResults:   totalResults,
			TotalPages:     totalPages,
		},
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling OKListResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}
