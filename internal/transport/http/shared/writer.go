package shared

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

type sourceError interface {
	FieldErrors() map[string]string
}

// Writer is used to send content to client using a HTTP connection.
type Writer struct {
	Logger *zerolog.Logger
}

// Response implements the protocol to be return from a HTTP response.
func (wrt Writer) Response(w http.ResponseWriter, r interface{}, status int, headers http.Header) {
	if headers != nil {
		for key, values := range headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	if r == nil {
		w.WriteHeader(status)
		return
	}

	content, err := json.Marshal(r)
	if err != nil {
		wrt.Logger.Err(err).Msg("error during response marshal")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	writed, err := w.Write(content)
	if err != nil {
		wrt.Logger.Err(err).Msg("error during write payload")
		return
	}
	if writed != len(content) {
		wrt.Logger.Error().Msgf(
			"invalid quantity of writed bytes, expected %d and got %d",
			len(content), writed,
		)
	}
}

// Error is used to generate a proper error content to be sent to the client.
func (wrt Writer) Error(w http.ResponseWriter, title string, err error, status int) {
	resp := struct {
		Error struct {
			Title  string            `json:"title"`
			Detail string            `json:"detail,omitempty"`
			Source map[string]string `json:"source,omitempty"`
		} `json:"error"`
	}{}
	if err != nil {
		resp.Error.Detail = err.Error()
	}
	if title != "" {
		resp.Error.Title = title
	}
	if serr, ok := err.(sourceError); ok {
		resp.Error.Source = serr.FieldErrors()
	}
	wrt.Response(w, &resp, status, nil)
}
