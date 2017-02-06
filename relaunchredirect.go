package relaunchredirect

import (
	"encoding/csv"
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type Redirect struct {
	ForceHost                  string
	ForceTLS                   bool
	ForceLowerCase             bool
	ForceLowerCaseIgnore       string
	ForceTrailingSlash         bool
	ForceTrailingSlashIgnore   string
	ForceNoTrailingSlash       bool
	ForceNoTrailingSlashIgnore string
	Redirects                  map[string]string
	RegexRedirects             map[string]string
}

// NewRedirect constructs a new Redirect
func NewRedirect() *Redirect {
	return &Redirect{
		ForceTLS:             false,
		ForceLowerCase:       false,
		ForceTrailingSlash:   false,
		ForceNoTrailingSlash: false,
		Redirects:            map[string]string{},
		RegexRedirects:       map[string]string{},
	}
}

// AppendRedirects load redirects with rules from a CSV File
func (r *Redirect) AppendRedirects(csvFile string) error {
	return r.appendCSV(csvFile, r.Redirects)
}

// AppendRegexRedirects load regex redirects with rules from a CSV File
func (r *Redirect) AppendRegexRedirects(csvFile string) error {
	return r.appendCSV(csvFile, r.RegexRedirects)
}

// ShouldRedirect tells you, if r needs to be redirected or not
func (r *Redirect) ShouldRedirect(req *http.Request) bool {

	// tls i.e. http://foo.com > https://foo.com
	if r.ForceTLS && req.TLS == nil {
		return true
	}

	// host i.e. foo.com > www.foo.com
	if len(r.ForceHost) > 0 {
		// check if we're being forwarded
		forwardedHost, ok := req.Header["X-Forwarded-Host"]
		if ok && len(forwardedHost) == 1 && len(forwardedHost[0]) > 0 {
			if forwardedHost[0] != r.ForceHost {
				return true
			}
		} else if req.URL.Host != r.ForceHost {
			return true
		}
	}

	// Lower case
	if r.shouldRedirectLowerCase(req) {
		return true
	}

	// Trailing slash
	if r.shouldRedirectTrailingSlash(req) {
		return true
	}

	// No trailing slash
	if r.shouldRedirectNoTrailingSlash(req) {
		return true
	}

	// configured regex redirects i.e. /de/some-url > /some-url
	if len(r.RegexRedirects) > 0 {
		for expression := range r.RegexRedirects {
			regex, err := regexp.Compile(expression)
			if err != nil {
				continue
			}
			if regex.MatchString(req.URL.Path) {
				return true
			}
		}
	}

	// configured redirects i.e. /some-url > /configured-url
	if _, ok := r.Redirects[req.URL.Path]; ok {
		return true
	}

	return false
}



func (r *Redirect) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// copy old url
	newURL, err := url.Parse(req.URL.String())
	if err != nil {
		// that should really never happen
		http.Error(w, "redirection error", http.StatusInternalServerError)
		return
	}

	// TLS
	if r.ForceTLS {
		newURL.Scheme = "https"
	}

	// Host
	if len(r.ForceHost) > 0 {
		newURL.Host = r.ForceHost
	}

	// Lower case
	if r.shouldRedirectLowerCase(req) {
		newURL.Path = strings.ToLower(newURL.Path)
	}

	// No / trailing slash
	if r.shouldRedirectTrailingSlash(req) {
		newURL.Path = newURL.Path + "/"
	} else if r.shouldRedirectNoTrailingSlash(req) {
		newURL.Path = newURL.Path[0 : len(req.URL.Path)-1]
	}

	// configured regex redirects i.e. /de/some-url > /some-url
	if len(r.RegexRedirects) > 0 {
		for expression, replacement := range r.RegexRedirects {
			regex, err := regexp.Compile(expression)
			if err != nil {
				http.Error(w, "redirection error", http.StatusInternalServerError)
				return
			}
			if regex.MatchString(newURL.Path) {
				newURL.Path = regex.ReplaceAllString(newURL.Path, replacement)
			}
		}
	}

	// configured redirects i.e. /some-url > /configured-url
	if value, ok := r.Redirects[newURL.Path]; ok {
		newURL.Path = value
	}

	http.Redirect(w, req, newURL.String(), http.StatusMovedPermanently)
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

// appendCSV load redirects with rules from a CSV File
func (r *Redirect) appendCSV(csvFile string, target map[string]string) error {
	if len(csvFile) == 0 {
		return nil
	}

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
		target[lineArray[0]] = lineArray[1]
	}
	return nil
}

// shouldRedirectLowerCase i.e. http://foo.com/Some/URL > http://foo.com/Some/url
func (r *Redirect) shouldRedirectLowerCase(req *http.Request) bool {
	if r.ForceLowerCase {
		ignore := false
		if len(r.ForceLowerCaseIgnore) > 0 {
			if regex, err := regexp.Compile(r.ForceLowerCaseIgnore); err == nil && regex.MatchString(req.URL.Path) {
				ignore = true
			}
		}
		if !ignore && strings.ToLower(req.URL.Path) != req.URL.Path {
			return true
		}
	}
	return false
}

// shouldRedirectTrailingSlash returns true if trailing slash redirect is required
// i.e. http://foo.com/some/url > http://foo.com/some/url/
func (r *Redirect) shouldRedirectTrailingSlash(req *http.Request) bool {
	if r.ForceTrailingSlash && req.URL.Path != "/" {
		ignore := false
		if len(r.ForceTrailingSlashIgnore) > 0 {
			if regex, err := regexp.Compile(r.ForceTrailingSlashIgnore); err == nil && regex.MatchString(req.URL.Path) {
				ignore = true
			}
		}
		if !ignore && req.URL.Path[len(req.URL.Path)-1:] != "/" {
			return true
		}
	}
	return false
}

// shouldRedirectNoTrailingSlash returns true if no trailing slash redirect is required
// i.e. http://foo.com/some/url/ > http://foo.com/some/url
func (r *Redirect) shouldRedirectNoTrailingSlash(req *http.Request) bool {
	if r.ForceNoTrailingSlash && req.URL.Path != "/" {
		ignore := false
		if len(r.ForceNoTrailingSlashIgnore) > 0 {
			if regex, err := regexp.Compile(r.ForceNoTrailingSlashIgnore); err == nil && regex.MatchString(req.URL.Path) {
				ignore = true
			}
		}
		if !ignore && req.URL.Path[len(req.URL.Path)-1:] == "/" {
			return true
		}
	}
	return false
}