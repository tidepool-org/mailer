package api

import (
	"github.com/gorilla/mux"
	"github.com/tidepool-org/mailer/templates"
	"go.uber.org/zap"
	"io/fs"
	"net/http"
)

func TemplateSourcesHandler() (http.Handler, error) {
	f, err := fs.Sub(templates.Sources, "sources")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(f)), nil
}

func RenderedTemplatesHandler(logger *zap.SugaredLogger, tmplts templates.Templates) (http.HandlerFunc, error) {
	return func (w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		template, ok := tmplts[templates.TemplateName(params["name"])]
		if !ok {
			w.WriteHeader(404)
			return
		}
		result, err := template.Execute(nil)
		if err != nil {
			w.WriteHeader(500)
			logger.Error(err)
			return
		}

		w.WriteHeader(200)
		w.Header().Set("content-type", "text/html")
		w.Write([]byte(result.Body))
	}, nil
}