package object

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/brinick/github/client"
	"github.com/brinick/github/client/cachedclient"
	"github.com/brinick/logging"
)

// ------------------------------------------------------------------

var (
	cachedClient *cachedclient.PickledCachedClient
	// HTTPClient   = cachedclient.NewClient()
	PageIterator = client.NewGithubPageIterator

	// Error returned by an iterator when there is no next page
	// in the pagination
	NoMorePages = fmt.Errorf("No more results pages")
)

// NotAvailable is just this constant...
const NotAvailable = "<n/a>"

func HTTPClient() *cachedclient.PickledCachedClient {
	if cachedClient == nil {
		cachedClient = cachedclient.NewClient()
	}

	return cachedClient
}

// ------------------------------------------------------------------
// Utility functions
// ------------------------------------------------------------------

// parseJSON parses the json data string into the
// value pointed to by the object.
func parseJSON(data string, object interface{}) error {
	err := json.Unmarshal([]byte(data), object)
	if err != nil {
		logging.Error(
			"Unable to parse JSON using given data type",
			logging.F("err", err),
			logging.F("type", reflect.TypeOf(object)),
		)

		logging.Error(data)
	}
	return err
}

// ------------------------------------------------------------------

func format(template string, values ...interface{}) string {
	return fmt.Sprintf(template, values...)
}

// ------------------------------------------------------------------

// join makes a path from the fragments passed
func join(bits ...string) string {
	return filepath.Join(bits...)
}

// ------------------------------------------------------------------

// stringVal returns a string that is either the default
// value or the de-referenced string pointer if the latter
// is not nil.
func stringVal(s *string, def string) string {
	val := def
	if s != nil {
		val = *s
	}
	return val
}

// ------------------------------------------------------------------
