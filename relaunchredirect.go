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
	ForceHost            string
	ForceTLS             bool
	ForceLowerCase       bool
	ForceTrailingSlash   bool
	ForceNoTrailingSlash bool
	Redirects            map[string]string
	RegexRedirects       map[string]string
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

	// case i.e. http://foo.com/Some/URL > http://foo.com/Some/URL
	if r.ForceLowerCase && strings.ToLower(req.URL.Path) != req.URL.Path {
		return true
	}

	// trailing slash i.e. http://foo.com/some/url/ > http://foo.com/some/url
	if r.ForceTrailingSlash && req.URL.Path != "/" && req.URL.Path[len(req.URL.Path)-1:] != "/" {
		return true
	} else if r.ForceNoTrailingSlash && req.URL.Path != "/" && req.URL.Path[len(req.URL.Path)-1:] == "/" {
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

	// tls i.e. http://foo.com > https://foo.com
	if r.ForceTLS {
		newURL.Scheme = "https"
	}

	// host i.e. foo.com > www.foo.com
	if len(r.ForceHost) > 0 {
		newURL.Host = r.ForceHost
	}

	// case i.e. http://foo.com/Some/URL > http://foo.com/Some/URL
	if r.ForceLowerCase {
		newURL.Path = strings.ToLower(newURL.Path)
	}

	// trailing slash i.e. http://foo.com/some/url/ > http://foo.com/some/url
	if r.ForceTrailingSlash && req.URL.Path != "/" && req.URL.Path[len(req.URL.Path)-1:] != "/" {
		newURL.Path = newURL.Path + "/"
	} else if r.ForceNoTrailingSlash && req.URL.Path != "/" && req.URL.Path[len(req.URL.Path)-1:] == "/" {
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
