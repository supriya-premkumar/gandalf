package types

import (
	"fmt"
	"time"

	admission "k8s.io/api/admission/v1"
)

// Config captures allowed labels. It enforces a whitelist model.
type Config struct {
	MatchLabels map[string]string `json:"match-labels"`
}

// APIResponse is a wrapper to send status codes from the rest API.
// While this is not the correct response that the admission controller must send to API Server. We use this to
// simplify local testing and debugging. The admission requests that are successfully evaluated will always send the
// correct response back to the API Server. We use this to easily debug errors.
type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"Message"`
}

const (
	// DefaultRESTPort is the listen address
	DefaultRESTPort = 8443

	// DefaultHTTPReadTimeout is the amount of time allowed to read request headers
	DefaultHTTPReadTimeout = time.Second * 15

	// DefaultHTTPWriteTimeout is the maximum duration before timing out writes of the response
	DefaultHTTPWriteTimeout = time.Second * 15

	// DefaultHTTPIdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled
	DefaultHTTPIdleTimeout = time.Second * 60

	// DefaultServerCrtPath is the default cert path. This cert needs to able to talk to API Server
	DefaultServerCrtPath = "/certs/server.crt"

	// DefaultServerKeyPath is the default tls key path
	DefaultServerKeyPath = "/certs/server-key.pem"

	// DefaultConfigPath is the default config file path. This has all the allowed labels
	DefaultConfigPath = "/gandalf-config.json"
)

// AdmissionReviewer is the interface. Implementers need to implement a single function.
// The rest handlers are appropriately plumbed
type AdmissionReviewer interface {
	Review(admissionReview *admission.AdmissionReview) (*admission.AdmissionResponse, error)
}

// RESTServer is the interface for fronting a REST API
type RESTServer interface {
	Start() error
	Stop()
}

// FixedWidthFormatter center alings the component. Custom formatter to align the components
func FixedWidthFormatter(component string) string {
	return fmt.Sprintf("%-12s", fmt.Sprintf("%12s", component))
}
