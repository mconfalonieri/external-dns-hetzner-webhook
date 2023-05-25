package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/plan"
	"github.com/ionos-cloud/external-dns-ionos-plugin/pkg/provider"
	log "github.com/sirupsen/logrus"
)

const (
	mediaTypeFormat        = "application/external.dns.plugin+json;"
	contentTypeHeader      = "Content-Type"
	contentTypePlaintext   = "text/plain"
	acceptHeader           = "Accept"
	varyHeader             = "Vary"
	supportedMediaVersions = "1"
	healthPath             = "/health"
	logFieldRequestPath    = "requestPath"
	logFieldRequestMethod  = "requestMethod"
	logFieldError          = "error"
)

type mediaType string

func mediaTypeVersion(v string) mediaType {
	return mediaType(mediaTypeFormat + "version=" + v)
}

func (m mediaType) Is(headerValue string) bool {
	return string(m) == headerValue
}

func checkAndSetMediaTypeHeaderValue(value string) error {
	for _, v := range strings.Split(supportedMediaVersions, ",") {
		if mediaTypeVersion(v).Is(value) {
			currentMediaType = mediaTypeVersion(v)
			return nil
		}
	}
	supportedMediaTypesString := ""
	for i, v := range strings.Split(supportedMediaVersions, ",") {
		sep := ""
		if i < len(supportedMediaVersions)-1 {
			sep = ", "
		}
		supportedMediaTypesString += string(mediaTypeVersion(v)) + sep
	}
	return fmt.Errorf("unsupported media type version: '%s'. Supported media types are: '%s'", value, supportedMediaTypesString)
}

func checkMediaTypeSet() error {
	if len(currentMediaType) == 0 {
		return fmt.Errorf("media type not set, please call the negotiation endpoint first")
	}
	return nil
}

var currentMediaType mediaType

// Plugin for external dns provider
type Plugin struct {
	provider provider.Provider
}

// New creates a new instance of the Plugin
func New(provider provider.Provider) *Plugin {
	p := Plugin{provider: provider}
	return &p
}

func (p *Plugin) hasAcceptHeader(w http.ResponseWriter, r *http.Request) bool {
	if len(r.Header.Get(acceptHeader)) == 0 {
		w.Header().Set(contentTypeHeader, contentTypePlaintext)
		w.WriteHeader(http.StatusNotAcceptable)
		err := fmt.Errorf("client must provide an accept header")
		fmt.Fprint(w, err.Error())
		requestLog(r).WithField(logFieldError, err).Info("accept header check failed")
		return false
	}
	return true
}

func (p *Plugin) hasContentHeader(w http.ResponseWriter, r *http.Request) bool {
	if len(r.Header.Get(contentTypeHeader)) == 0 {
		w.Header().Set(contentTypeHeader, contentTypePlaintext)
		w.WriteHeader(http.StatusNotAcceptable)
		err := fmt.Errorf("client must provide a content type")
		fmt.Fprint(w, err.Error())
		requestLog(r).WithField(logFieldError, err).Info("content type header check failed")
		return false
	}
	return true
}

func Health(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) AcceptType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			accept := r.Header.Get(acceptHeader)
			if len(accept) > 0 && !currentMediaType.Is(accept) {
				if err := checkMediaTypeSet(); err != nil {
					w.WriteHeader(http.StatusNotAcceptable)
					fmt.Fprint(w, err.Error())
					requestLog(r).WithField(logFieldError, err).Info("accept header check failed")
					return
				}
				w.Header().Set(contentTypeHeader, contentTypePlaintext)
				w.WriteHeader(http.StatusUnsupportedMediaType)
				err := fmt.Errorf("only allows media type '%s' as accept header ", currentMediaType)
				fmt.Fprint(w, err.Error())
				requestLog(r).WithField(logFieldError, err).Info("accept header check failed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) ContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			contentType := r.Header.Get(contentTypeHeader)
			if len(contentType) > 0 && !currentMediaType.Is(contentType) {
				if err := checkMediaTypeSet(); err != nil {
					w.WriteHeader(http.StatusNotAcceptable)
					fmt.Fprint(w, err.Error())
					requestLog(r).WithField(logFieldError, err).Info("content type header check failed")
					return
				}
				w.Header().Set(contentTypeHeader, contentTypePlaintext)
				w.WriteHeader(http.StatusUnsupportedMediaType)
				err := fmt.Errorf("only allows media type '%s' as content-type", currentMediaType)
				fmt.Fprint(w, err.Error())
				requestLog(r).WithField(logFieldError, err).Info("content type header check failed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Negotiate get request endpoint handler
func (p *Plugin) Negotiate(w http.ResponseWriter, r *http.Request) {
	if p.hasAcceptHeader(w, r) {
		err := checkAndSetMediaTypeHeaderValue(r.Header.Get(acceptHeader))
		if err != nil {
			w.Header().Set(contentTypeHeader, contentTypePlaintext)
			w.WriteHeader(http.StatusUnsupportedMediaType)
			fmt.Fprint(w, err.Error())
			requestLog(r).Info(err.Error())
			return
		}
		w.Header().Set(varyHeader, contentTypeHeader)
		w.Header().Set(contentTypeHeader, string(currentMediaType))
		requestLog(r).Debugf("negotiating media type, returning media type: '%s'", currentMediaType)
		w.WriteHeader(http.StatusOK)
	}
}

// Records get request endpoint handler
func (p *Plugin) Records(w http.ResponseWriter, r *http.Request) {
	if p.hasAcceptHeader(w, r) {
		requestLog(r).Debug("requesting records")
		ctx := r.Context()
		records, err := p.provider.Records(ctx)
		if err != nil {
			requestLog(r).WithField(logFieldError, err).Error("error getting records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(contentTypeHeader, string(currentMediaType))
		w.WriteHeader(http.StatusOK)
		requestLog(r).Debugf("returning records count: %d", len(records))
		err = json.NewEncoder(w).Encode(records)
		if err != nil {
			requestLog(r).WithField(logFieldError, err).Error("error encoding records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// ApplyChanges get request endpoint handler
func (p *Plugin) ApplyChanges(w http.ResponseWriter, r *http.Request) {
	if p.hasContentHeader(w, r) {
		var changes plan.Changes
		ctx := r.Context()
		if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
			w.Header().Set(contentTypeHeader, contentTypePlaintext)
			w.WriteHeader(http.StatusBadRequest)
			errMsg := fmt.Sprintf("error decoding changes: %s", err.Error())
			fmt.Fprint(w, errMsg)
			requestLog(r).WithField(logFieldError, err).Info(errMsg)
			return
		}
		requestLog(r).Debugf("requesting apply changes, create: %d , updateOld: %d, updateNew: %d, delete: %d",
			len(changes.Create), len(changes.UpdateOld), len(changes.UpdateNew), len(changes.Delete))
		err := p.provider.ApplyChanges(ctx, &changes)
		if err != nil {
			w.Header().Set(contentTypeHeader, contentTypePlaintext)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// PropertyValuesEqualsRequest holds params for property values equals request
type PropertyValuesEqualsRequest struct {
	Name     string `json:"name"`
	Previous string `json:"previous"`
	Current  string `json:"current"`
}

// PropertiesValuesEqualsResponse holds params for property values equals response
type PropertiesValuesEqualsResponse struct {
	Equals bool `json:"equals"`
}

// PropertyValuesEquals get request endpoint handler
func (p *Plugin) PropertyValuesEquals(w http.ResponseWriter, r *http.Request) {
	if p.hasContentHeader(w, r) && p.hasAcceptHeader(w, r) {
		pve := PropertyValuesEqualsRequest{}
		if err := json.NewDecoder(r.Body).Decode(&pve); err != nil {
			w.Header().Set(contentTypeHeader, contentTypePlaintext)
			w.WriteHeader(http.StatusBadRequest)
			errMessage := fmt.Sprintf("failed to decode request body: %v", err)
			fmt.Fprint(w, errMessage)
			requestLog(r).WithField(logFieldError, err).Info(errMessage)
			return
		}
		requestLog(r).Debugf("requesting property values equals, name: %s, previous: %s , current: %s",
			pve.Name, pve.Previous, pve.Current)
		valuesEqual := p.provider.PropertyValuesEqual(pve.Name, pve.Previous, pve.Current)
		resp := PropertiesValuesEqualsResponse{
			Equals: valuesEqual,
		}
		out, _ := json.Marshal(&resp)
		requestLog(r).Debugf("return property values equals response equals: %v", valuesEqual)
		w.Header().Set(contentTypeHeader, string(currentMediaType))
		fmt.Fprint(w, string(out))
	}
}

// AdjustEndpoints get request endpoint handler
func (p *Plugin) AdjustEndpoints(w http.ResponseWriter, r *http.Request) {
	if p.hasContentHeader(w, r) && p.hasAcceptHeader(w, r) {
		var pve []*endpoint.Endpoint
		if err := json.NewDecoder(r.Body).Decode(&pve); err != nil {
			w.Header().Set(contentTypeHeader, contentTypePlaintext)
			w.WriteHeader(http.StatusBadRequest)
			errMessage := fmt.Sprintf("failed to decode request body: %v", err)
			log.Infof(errMessage+" , request method: %s, request path: %s", r.Method, r.URL.Path)
			fmt.Fprint(w, errMessage)
			return
		}
		log.Debugf("requesting adjust endpoints count: %d", len(pve))
		pve = p.provider.AdjustEndpoints(pve)
		out, _ := json.Marshal(&pve)
		log.Debugf("return adjust endpoints response, resultEndpointCount: %d", len(pve))
		w.Header().Set(contentTypeHeader, string(currentMediaType))
		fmt.Fprint(w, string(out))
	}
}

func requestLog(r *http.Request) *log.Entry {
	return log.WithFields(log.Fields{logFieldRequestMethod: r.Method, logFieldRequestPath: r.URL.Path})
}
