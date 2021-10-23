package multipartestutils

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
)

// Builder is a multipart builder.
// It is not thread-safe.
type Builder struct {
	pageURL string
	eb      eventBody
	cbs     []func(*multipart.Writer) error
}

// New constructs new multipart Builder.
func NewMultipartBuilder() *Builder {
	return &Builder{}
}

// AddField adds multipart field.
func (b *Builder) AddField(fieldName, value string) *Builder {
	b.cbs = append(b.cbs, func(mw *multipart.Writer) error {
		if err := mw.WriteField(fieldName, value); err != nil {
			return fmt.Errorf("multipartbuilder: failed to write field %s=%s: %s", fieldName, value, err.Error())
		}
		return nil
	})
	return b
}

// AddReader adds multipart file field from provided reader.
func (b *Builder) AddReader(fieldName, fileName string, reader io.Reader) *Builder {
	b.cbs = append(b.cbs, func(mw *multipart.Writer) error {

		w, err := mw.CreateFormFile(fieldName, fileName)
		if err != nil {
			return fmt.Errorf("multipartbuilder: failed to create form file %s (%s) for reader: %s", fieldName, fileName, err.Error())
		}

		_, err = io.Copy(w, reader)
		if err != nil {
			return fmt.Errorf("multipartbuilder: failed to copy form file %s (%s) for reader: %s", fieldName, fileName, err.Error())
		}

		return nil
	})
	return b
}

// AddFile adds multipart file field from specified file path.
func (b *Builder) AddFile(fieldName, filePath string) *Builder {
	b.cbs = append(b.cbs, func(mw *multipart.Writer) error {

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("multipartbuilder: failed to open file %s (%s): %s", fieldName, filePath, err.Error())
		}
		defer f.Close()

		w, err := mw.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			return fmt.Errorf("multipartbuilder: failed to create form file %s (%s): %s", fieldName, filePath, err.Error())
		}

		_, err = io.Copy(w, f)
		if err != nil {
			return fmt.Errorf("multipartbuilder: failed to copy form file %s (%s): %s", fieldName, filePath, err.Error())
		}

		return nil
	})
	return b
}

// Build finalizes Builder, returning Content-Type and multipart reader.
// It should be called only once for Builder.
// Returned reader should be used (Read/Close) at least once to clean up properly.
// Any errors are bound to returned reader (will be returned on Read/Close).
func (b *Builder) Build() (string, io.ReadCloser) {
	r, w := io.Pipe()
	mw := multipart.NewWriter(w)

	go func() {
		for _, cb := range b.cbs {
			if err := cb(mw); err != nil {
				_ = w.CloseWithError(err)
				return
			}
		}
		_ = w.CloseWithError(mw.Close())
	}()

	return mw.FormDataContentType(), r
}

type EventFuncID struct {
	ID     string   `json:"id,omitempty"`
	Params []string `json:"params,omitempty"`
}

type Event struct {
	Checked bool   `json:"checked,omitempty"` // For Checkbox
	From    string `json:"from,omitempty"`    // For DatePicker
	To      string `json:"to,omitempty"`      // For DatePicker
	Value   string `json:"value,omitempty"`   // For Input, DatePicker
}

type eventBody struct {
	EventFuncID EventFuncID `json:"eventFuncId,omitempty"`
	Event       Event       `json:"event,omitempty"`
}

func (b *Builder) EventFunc(id string, params ...string) *Builder {
	b.eb.EventFuncID.ID = id
	b.eb.EventFuncID.Params = params
	return b
}

func (b *Builder) Event(evt Event) *Builder {
	b.eb.Event = evt
	return b
}

func (b *Builder) PageURL(url string) *Builder {
	b.pageURL = url
	return b
}

func (b *Builder) BuildEventFuncRequest() (r *http.Request) {
	eventBody, _ := json.Marshal(b.eb)
	b.AddField("__event_data__", string(eventBody))

	contentType, body := b.Build()
	pu := b.pageURL
	if len(b.pageURL) == 0 {
		pu = "/"
	}
	r = httptest.NewRequest("POST", fmt.Sprintf("%s?__execute_event__=%s", pu, b.eb.EventFuncID.ID), body)
	r.Header.Add("Content-Type", contentType)
	return
}
