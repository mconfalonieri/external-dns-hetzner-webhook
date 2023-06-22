package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/endpoint"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/plan"

	"github.com/ionos-cloud/external-dns-ionos-webhook/cmd/webhook/init/configuration"
	"github.com/ionos-cloud/external-dns-ionos-webhook/pkg/webhook"
)

type testCase struct {
	name                                string
	returnRecords                       []*endpoint.Endpoint
	returnAdjustedEndpoints             []*endpoint.Endpoint
	hasError                            error
	method                              string
	path                                string
	headers                             map[string]string
	body                                string
	expectedStatusCode                  int
	expectedResponseHeaders             map[string]string
	expectedBody                        string
	expectedChanges                     *plan.Changes
	expectedEndpointsToAdjust           []*endpoint.Endpoint
	expectedPropertyValuesEqualName     string
	expectedPropertyValuesEqualPrevious string
	expectedPropertyValuesEqualCurrent  string
}

var mockProvider *MockProvider

func TestMain(m *testing.M) {
	mockProvider = &MockProvider{}
	srv := Init(configuration.Init(), webhook.New(mockProvider))
	go ShutdownGracefully(srv)
	time.Sleep(300 * time.Millisecond)
	m.Run()
	if err := srv.Shutdown(context.TODO()); err != nil {
		panic(err)
	}
}

func TestRecords(t *testing.T) {
	testCases := []testCase{
		{
			name: "happy case",
			returnRecords: []*endpoint.Endpoint{
				{
					DNSName:    "test.example.com",
					Targets:    []string{""},
					RecordType: "A",
					RecordTTL:  3600,
					Labels: map[string]string{
						"label1": "value1",
					},
				},
			},
			method:             http.MethodGet,
			headers:            map[string]string{"Accept": "application/external.dns.webhook+json;version=1"},
			path:               "/records",
			body:               "",
			expectedStatusCode: http.StatusOK,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
			},
			expectedBody: "[{\"dnsName\":\"test.example.com\",\"targets\":[\"\"],\"recordType\":\"A\",\"recordTTL\":3600,\"labels\":{\"label1\":\"value1\"}}]",
		},
		{
			name:               "no accept header",
			method:             http.MethodGet,
			headers:            map[string]string{},
			path:               "/records",
			body:               "",
			expectedStatusCode: http.StatusNotAcceptable,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide an accept header",
		},
		{
			name:               "wrong accept header",
			method:             http.MethodGet,
			headers:            map[string]string{"Accept": "invalid"},
			path:               "/records",
			body:               "",
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a valid versioned media type in the accept header: unsupported media type version: 'invalid'. Supported media types are: 'application/external.dns.webhook+json;version=1'",
		},
		{
			name:               "backend error",
			hasError:           fmt.Errorf("backend error"),
			method:             http.MethodGet,
			headers:            map[string]string{"Accept": "application/external.dns.webhook+json;version=1"},
			path:               "/records",
			body:               "",
			expectedStatusCode: http.StatusInternalServerError,
		},
	}
	executeTestCases(t, testCases)
}

func TestApplyChanges(t *testing.T) {
	testCases := []testCase{
		{
			name:   "happy case",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
			},
			path: "/records",
			body: `
{
    "Create": [
        {
            "dnsName": "test.example.com",
            "targets": ["11.11.11.11"],
            "recordType": "A",
            "recordTTL": 3600,
            "labels": {
                "label1": "value1",
                "label2": "value2"
            }
        }
    ]
}`,
			expectedStatusCode:      http.StatusNoContent,
			expectedResponseHeaders: map[string]string{},
			expectedBody:            "",
			expectedChanges: &plan.Changes{
				Create: []*endpoint.Endpoint{
					{
						DNSName:    "test.example.com",
						Targets:    []string{"11.11.11.11"},
						RecordType: "A",
						RecordTTL:  3600,
						Labels: map[string]string{
							"label1": "value1",
							"label2": "value2",
						},
					},
				},
			},
		},
		{
			name:               "no content type header",
			method:             http.MethodPost,
			path:               "/records",
			body:               "",
			expectedStatusCode: http.StatusNotAcceptable,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a content type",
		},
		{
			name:   "wrong content type header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "invalid",
			},
			path:               "/records",
			body:               "",
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a valid versioned media type in the content type: unsupported media type version: 'invalid'. Supported media types are: 'application/external.dns.webhook+json;version=1'",
		},
		{
			name:   "invalid json",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path:               "/records",
			body:               "invalid",
			expectedStatusCode: http.StatusBadRequest,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "error decoding changes: invalid character 'i' looking for beginning of value",
		},
		{
			name:     "backend error",
			hasError: fmt.Errorf("backend error"),
			method:   http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path: "/records",
			body: `
{
    "Create": [
        {
            "dnsName": "test.example.com",
            "targets": ["11.11.11.11"],
            "recordType": "A",
            "recordTTL": 3600,
            "labels": {
                "label1": "value1",
                "label2": "value2"
            }
        }
    ]
}`,
			expectedStatusCode: http.StatusInternalServerError,
		},
	}
	executeTestCases(t, testCases)
}

func TestAdjustEndpoints(t *testing.T) {
	testCases := []testCase{
		{
			name: "happy case",
			returnAdjustedEndpoints: []*endpoint.Endpoint{
				{
					DNSName:    "adjusted.example.com",
					Targets:    []string{""},
					RecordType: "A",
					RecordTTL:  3600,
					Labels: map[string]string{
						"label1": "value1",
					},
				},
			},
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path: "/adjustendpoints",
			body: `
[
	{
		"dnsName": "toadjust.example.com",
		"targets": [],
		"recordType": "A",
		"recordTTL": 3600,
		"labels": {
			"label1": "value1",
			"label2": "value2"
		}
	}
]`,
			expectedStatusCode: http.StatusOK,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
			},
			expectedBody: "[{\"dnsName\":\"adjusted.example.com\",\"targets\":[\"\"],\"recordType\":\"A\",\"recordTTL\":3600,\"labels\":{\"label1\":\"value1\"}}]",
			expectedEndpointsToAdjust: []*endpoint.Endpoint{
				{
					DNSName:    "toadjust.example.com",
					Targets:    []string{},
					RecordType: "A",
					RecordTTL:  3600,
					Labels: map[string]string{
						"label1": "value1",
						"label2": "value2",
					},
				},
			},
		},
		{
			name:   "no content type header",
			method: http.MethodPost,
			headers: map[string]string{
				"Accept": "application/external.dns.webhook+json;version=1",
			},
			path:               "/adjustendpoints",
			body:               "",
			expectedStatusCode: http.StatusNotAcceptable,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a content type",
		},
		{
			name:   "wrong content type header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "invalid",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path:               "/adjustendpoints",
			body:               "",
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a valid versioned media type in the content type: unsupported media type version: 'invalid'. Supported media types are: 'application/external.dns.webhook+json;version=1'",
		},
		{
			name:   "no accept header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
			},
			path:               "/adjustendpoints",
			body:               "",
			expectedStatusCode: http.StatusNotAcceptable,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide an accept header",
		},
		{
			name:   "wrong accept header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "invalid",
			},
			path:               "/adjustendpoints",
			body:               "",
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a valid versioned media type in the accept header: unsupported media type version: 'invalid'. Supported media types are: 'application/external.dns.webhook+json;version=1'",
		},
		{
			name:   "invalid json",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path:               "/adjustendpoints",
			body:               "invalid",
			expectedStatusCode: http.StatusBadRequest,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "failed to decode request body: invalid character 'i' looking for beginning of value",
		},
	}
	executeTestCases(t, testCases)
}

func TestPropertyValuesEqual(t *testing.T) {
	testCases := []testCase{
		{
			name:   "happy case",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path: "/propertyvaluesequals",
			body: `
{
	"name": "propertyname",
	"previous": "previousvalue",
	"current": "currentvalue"
}`,
			expectedStatusCode: http.StatusOK,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
			},
			expectedBody:                        "{\"equals\":false}",
			expectedPropertyValuesEqualName:     "propertyname",
			expectedPropertyValuesEqualCurrent:  "currentvalue",
			expectedPropertyValuesEqualPrevious: "previousvalue",
		},
		{
			name:   "no content type header",
			method: http.MethodPost,
			headers: map[string]string{
				"Accept": "application/external.dns.webhook+json;version=1",
			},
			path:               "/propertyvaluesequals",
			body:               "",
			expectedStatusCode: http.StatusNotAcceptable,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a content type",
		},
		{
			name:   "wrong content type header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "invalid",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path:               "/propertyvaluesequals",
			body:               "",
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a valid versioned media type in the content type: unsupported media type version: 'invalid'. Supported media types are: 'application/external.dns.webhook+json;version=1'",
		},
		{
			name:   "no accept header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
			},
			path:               "/propertyvaluesequals",
			body:               "",
			expectedStatusCode: http.StatusNotAcceptable,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide an accept header",
		},
		{
			name:   "wrong accept header",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "invalid",
			},
			path:               "/propertyvaluesequals",
			body:               "",
			expectedStatusCode: http.StatusUnsupportedMediaType,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "client must provide a valid versioned media type in the accept header: unsupported media type version: 'invalid'. Supported media types are: 'application/external.dns.webhook+json;version=1'",
		},
		{
			name:   "invalid json",
			method: http.MethodPost,
			headers: map[string]string{
				"Content-Type": "application/external.dns.webhook+json;version=1",
				"Accept":       "application/external.dns.webhook+json;version=1",
			},
			path:               "/propertyvaluesequals",
			body:               "invalid",
			expectedStatusCode: http.StatusBadRequest,
			expectedResponseHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
			expectedBody: "failed to decode request body: invalid character 'i' looking for beginning of value",
		},
	}
	executeTestCases(t, testCases)
}

func executeTestCases(t *testing.T, testCases []testCase) {
	log.SetLevel(log.DebugLevel)
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d. %s", i+1, tc.name), func(t *testing.T) {
			mockProvider.testCase = tc
			mockProvider.t = t
			var bodyReader io.Reader
			if tc.body != "" {
				bodyReader = strings.NewReader(tc.body)
			}
			request, err := http.NewRequest(tc.method, "http://localhost:8888"+tc.path, bodyReader)
			if err != nil {
				t.Error(err)
			}
			for k, v := range tc.headers {
				request.Header.Set(k, v)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Error(err)
			}
			if response.StatusCode != tc.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tc.expectedStatusCode, response.StatusCode)
			}
			for k, v := range tc.expectedResponseHeaders {
				if response.Header.Get(k) != v {
					t.Errorf("expected header '%s' with value '%s', got '%s'", k, v, response.Header.Get(k))
				}
			}
			if tc.expectedBody != "" {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Error(err)
				}
				_ = response.Body.Close()
				actualTrimmedBody := strings.TrimSpace(string(body))
				if actualTrimmedBody != tc.expectedBody {
					t.Errorf("expected body '%s', got '%s'", tc.expectedBody, actualTrimmedBody)
				}
			}
		})
	}
}

type MockProvider struct {
	t        *testing.T
	testCase testCase
}

// Records MockProvider implementation to be removed when real providers are added
func (d *MockProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	return d.testCase.returnRecords, d.testCase.hasError
}

// ApplyChanges MockProvider implementation to be removed when real providers are added
func (d *MockProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if d.testCase.hasError != nil {
		return d.testCase.hasError
	}
	if !reflect.DeepEqual(changes, d.testCase.expectedChanges) {
		d.t.Errorf("expected changes '%v', got '%v'", d.testCase.expectedChanges, changes)
	}
	return nil
}

func (d *MockProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) []*endpoint.Endpoint {
	if !reflect.DeepEqual(endpoints, d.testCase.expectedEndpointsToAdjust) {
		d.t.Errorf("expected endpoints to adjust '%v', got '%v'", d.testCase.expectedEndpointsToAdjust, endpoints)
	}
	return d.testCase.returnAdjustedEndpoints
}

func (d *MockProvider) PropertyValuesEqual(name string, previous string, current string) bool {
	if name != d.testCase.expectedPropertyValuesEqualName {
		d.t.Errorf("expected name '%s', got '%s'", d.testCase.expectedPropertyValuesEqualName, name)
	}
	if previous != d.testCase.expectedPropertyValuesEqualPrevious {
		d.t.Errorf("expected previous '%s', got '%s'", d.testCase.expectedPropertyValuesEqualPrevious, previous)
	}
	if current != d.testCase.expectedPropertyValuesEqualCurrent {
		d.t.Errorf("expected current '%s', got '%s'", d.testCase.expectedPropertyValuesEqualCurrent, current)
	}
	return previous == current
}
