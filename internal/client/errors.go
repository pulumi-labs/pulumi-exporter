package client

import (
	"errors"
	"fmt"
	"time"
)

const (
	// requestTimeout bounds every HTTP request to the Pulumi Cloud API.
	requestTimeout = 30 * time.Second
	// defaultPageSize is the page size requested from paginated endpoints.
	defaultPageSize = 100
)

var (
	// ErrUnexpectedStatus is returned when the Pulumi Cloud API answers with a
	// status code other than the one the operation expects.
	ErrUnexpectedStatus = errors.New("unexpected status")
	// ErrFeatureNotEnabled is returned when the Pulumi Cloud API reports a
	// feature (Neo, Agents, Delivery) as unavailable for an organization.
	ErrFeatureNotEnabled = errors.New("feature not enabled for organization")
	// ErrInvalidAPIURL is returned when the configured API URL is not an
	// absolute URL with a host.
	ErrInvalidAPIURL = errors.New("api url must be absolute, e.g. https://api.pulumi.com")
)

func unexpectedStatus(op string, code int) error {
	return fmt.Errorf("%s: %w %d", op, ErrUnexpectedStatus, code)
}

func featureNotEnabled(op string) error {
	return fmt.Errorf("%s: %w", op, ErrFeatureNotEnabled)
}
