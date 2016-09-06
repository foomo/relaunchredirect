package relaunchredirect

import (
	"encoding/csv"
	"errors"
	"net/http"
	"net/url"
	"os"
)

type Redirect struct {
	Redirects map[string]string
	ForceHost string
	ForceTLS  bool
}

// NewRedirect constructs a new Redirect
func NewRedirect() *Redirect {
	return &Redirect{
		Redirects: map[string]string{},
	}
}

// AppendRedirects load redirects with rules from a CSV File
func (r *Redirect) AppendRedirects(csvFile string) error {
	f, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	confArray, csvErr := csvReader.ReadAll()
	if csvErr != nil {
		return csvErr
	}
	for _, lineArray := range confArray {
		if len(lineArray) != 2 {
			return errors.New("invalid redirects file")
		}
		r.Redirects[lineArray[0]] = lineArray[1]
	}
	return nil
}

// ShouldRedirect tells you, if r needs to be redirected or not
func (r *Redirect) ShouldRedirect(req *http.Request) bool {
	// path redirection ?
	_, pathRedirection := r.Redirects[req.URL.Path]
	if pathRedirection {
		return true
	}
	// tls
	if r.ForceTLS && req.TLS == nil {
		return true
	}
	// host
	if len(r.ForceHost) > 0 {
		return req.URL.Host != r.ForceHost
	}
	return false
}

func (r *Redirect) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path, pathRedirect := r.Redirects[req.URL.Path]
	if !pathRedirect {
		path = req.URL.Path
	}
	// copy old url
	newURL, err := url.Parse(req.URL.String())
	if err != nil {
		// that should really never happen
		http.Error(w, "redirection error", http.StatusInternalServerError)
		return
	}
	newURL.Path = path
	if r.ForceTLS {
		newURL.Scheme = "https"
	}
	if len(r.ForceHost) > 0 {
		newURL.Host = r.ForceHost
	}
	http.Redirect(w, req, newURL.String(), http.StatusMovedPermanently)
}
