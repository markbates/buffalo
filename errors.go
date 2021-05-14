package buffalo

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/gobuffalo/buffalo/internal/defaults"
	"github.com/gobuffalo/buffalo/internal/httpx"
	"github.com/gobuffalo/buffalo/internal/takeon/github.com/markbates/errx"
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/plush/v4"
)

// HTTPError a typed error returned by http Handlers and used for choosing error handlers
type HTTPError struct {
	Status int   `json:"status"`
	Cause  error `json:"error"`
}

func (h HTTPError) Error() string {
	return h.Cause.Error()
}

// ErrorHandler interface for handling an error for a
// specific status code.
type ErrorHandler func(int, error, Context) error

// ErrorHandlers is used to hold a list of ErrorHandler
// types that can be used to handle specific status codes.
/*
	a.ErrorHandlers[http.StatusInternalServerError] = func(status int, err error, c buffalo.Context) error {
		res := c.Response()
		res.WriteHeader(status)
		res.Write([]byte(err.Error()))
		return nil
	}
*/
type ErrorHandlers map[int]ErrorHandler

// Get a registered ErrorHandler for this status code. If
// no ErrorHandler has been registered, a default one will
// be returned.
func (e ErrorHandlers) Get(status int) ErrorHandler {
	if eh, ok := e[status]; ok {
		return eh
	}
	if eh, ok := e[0]; ok {
		return eh
	}
	return defaultErrorHandler
}

// Default sets an error handler should a status
// code not already be mapped. This will replace
// the original default error handler.
// This is a *catch-all* handler.
func (e ErrorHandlers) Default(eh ErrorHandler) {
	if eh != nil {
		e[0] = eh
	}
}

// PanicHandler recovers from panics gracefully and calls
// the error handling code for a 500 error.
func (a *App) PanicHandler(next Handler) Handler {
	return func(c Context) error {
		defer func() { //catch or finally
			r := recover()
			var err error
			if r != nil { //catch
				switch t := r.(type) {
				case error:
					err = t
				case string:
					err = fmt.Errorf(t)
				default:
					err = fmt.Errorf(fmt.Sprint(t))
				}
				events.EmitError(events.ErrPanic, err,
					map[string]interface{}{
						"context":    c,
						"app":        a,
						"stacktrace": string(debug.Stack()),
						"error":      err,
					},
				)
				eh := a.ErrorHandlers.Get(http.StatusInternalServerError)
				eh(http.StatusInternalServerError, err, c)
			}
		}()
		return next(c)
	}
}

func (a *App) defaultErrorMiddleware(next Handler) Handler {
	return func(c Context) error {
		err := next(c)
		if err == nil {
			return nil
		}
		status := http.StatusInternalServerError
		// unpack root cause and check for HTTPError
		cause := errx.Unwrap(err)
		switch cause {
		case sql.ErrNoRows:
			status = http.StatusNotFound
		default:
			if h, ok := cause.(HTTPError); ok {
				status = h.Status
			}
		}
		payload := events.Payload{
			"context": c,
			"app":     a,
		}
		events.EmitError(events.ErrGeneral, err, payload)

		eh := a.ErrorHandlers.Get(status)
		err = eh(status, err, c)
		if err != nil {
			events.Emit(events.Event{
				Kind:    EvtFailureErr,
				Message: "unable to handle error and giving up",
				Error:   err,
				Payload: payload,
			})
			// things have really hit the fan if we're here!!
			a.Logger.Error(err)
			c.Response().WriteHeader(http.StatusInternalServerError)
			c.Response().Write([]byte(err.Error()))
		}
		return nil
	}
}

func productionErrorResponseFor(status int) []byte {
	if status == http.StatusNotFound {
		return []byte(prodNotFoundTmpl)
	}

	return []byte(prodErrorTmpl)
}

// ErrorResponse is a used to display errors as JSON or XML
type ErrorResponse struct {
	XMLName xml.Name `json:"-" xml:"response"`
	Error   string   `json:"error" xml:"error"`
	Trace   string   `json:"trace" xml:"trace"`
	Code    int      `json:"code" xml:"code,attr"`
}

const defaultErrorCT = "text/html; charset=utf-8"

func defaultErrorHandler(status int, origErr error, c Context) error {
	env := c.Value("env")
	requestCT := defaults.String(httpx.ContentType(c.Request()), defaultErrorCT)

	c.LogField("status", status)
	c.Logger().Error(origErr)

	if env != nil && env.(string) == "production" {
		c.(*DefaultContext).writeHeaders(defaultErrorCT, status)
		responseBody := productionErrorResponseFor(status)
		c.Response().Write(responseBody)
		return nil
	}

	trace := origErr.Error()

	switch strings.ToLower(requestCT) {
	case "application/json", "text/json", "json":
		c.(*DefaultContext).writeHeaders("application/json", status)
		err := json.NewEncoder(c.Response()).Encode(&ErrorResponse{
			Error: errx.Unwrap(origErr).Error(),
			Trace: trace,
			Code:  status,
		})
		if err != nil {
			return err
		}
	case "application/xml", "text/xml", "xml":
		c.(*DefaultContext).writeHeaders("text/xml", status)
		err := xml.NewEncoder(c.Response()).Encode(&ErrorResponse{
			Error: errx.Unwrap(origErr).Error(),
			Trace: trace,
			Code:  status,
		})
		if err != nil {
			return err
		}
	default:
		c.(*DefaultContext).writeHeaders(defaultErrorCT, status)
		if err := c.Request().ParseForm(); err != nil {
			trace = fmt.Sprintf("%s\n%s", err.Error(), trace)
		}

		routes := c.Value("routes")
		cd := c.Data()

		delete(cd, "app")
		delete(cd, "routes")

		data := map[string]interface{}{
			"routes":      routes,
			"error":       trace,
			"status":      status,
			"data":        cd,
			"params":      c.Params(),
			"posted_form": c.Request().Form,
			"context":     c,
			"headers":     inspectHeaders(c.Request().Header),
			"inspect": func(v interface{}) string {
				return fmt.Sprintf("%+v", v)
			},
		}

		ctx := plush.NewContextWith(data)
		t, err := plush.Render(devErrorTmpl, ctx)
		if err != nil {
			return err
		}

		res := c.Response()
		_, err = res.Write([]byte(t))

		return err
	}
	return nil
}

func (c *DefaultContext) writeHeaders(contentType string, status int) {
	c.Response().Header().Set("content-type", contentType)
	c.Response().WriteHeader(status)
}

type inspectHeaders http.Header

func (i inspectHeaders) String() string {

	bb := make([]string, 0, len(i))

	for k, v := range i {
		bb = append(bb, fmt.Sprintf("%s: %s", k, v))
	}
	sort.Strings(bb)
	return strings.Join(bb, "\n\n")
}
