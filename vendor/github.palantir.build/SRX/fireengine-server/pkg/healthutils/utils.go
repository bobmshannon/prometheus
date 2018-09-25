package healthutils

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.palantir.build/deployability/voodoo/conjure/sls/spec/health"
)

// HealthyCheckHandler returns an HTTP handler that returns a healthy SLS health check
// endpoint response. Validates that the bearer token provided in the request matches
// the specified checkSecret.
func HealthyCheckHandler(checkType, checkMsg, checkSecret string) http.HandlerFunc {
	healthyCheckResult := health.HealthCheckResult{
		State:   health.HealthStateHealthy,
		Type:    health.CheckType(checkType),
		Message: &checkMsg,
	}
	healthCheckResponse := struct {
		Checks map[string]health.HealthCheckResult `json:"checks"`
	}{
		Checks: map[string]health.HealthCheckResult{
			checkType: healthyCheckResult,
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validateBearerToken(r, checkSecret) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp, err := json.Marshal(healthCheckResponse)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, string(resp))
		}
	})
}

// validateBearerToken returns true if the token in the request matches what is expected,
// and false otherwise.
func validateBearerToken(req *http.Request, expectedToken string) bool {
	if expectedToken == "" {
		return true
	}
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}
	parsedAuthHeader := strings.Split(authHeader, " ")
	if len(parsedAuthHeader) != 2 && parsedAuthHeader[0] != "Bearer" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(parsedAuthHeader[1]), []byte(expectedToken)) == 1 {
		return true
	}
	return false
}
