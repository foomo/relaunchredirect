# relaunchredirect

Helps with redirects, when relaunching a website:

- redirects paths through a map lookup
- redirects paths through a regex map lookup
- can force lower case only
- can force trailing slash
- can force no trailing slash
- can force a domain
- can force TLS

## Usage

Instantiation and configuration:

```go

r := relaunchredirect.NewRedirect()

// force tls
r.ForceTLS = true

// force lower case
r.ForceLowerCase = true

// force no or trailing slash
r.ForceTrailingSlash = true
// r.ForceNoTrailingSlash = true

// force a host
r.ForceHost = "example.com"

// set a redirect programatically
r.Redirects["/from"] = "/to"

// append redirects from a CSV
if err := r.AppendRedirects("/path/to/my/redirects.csv") er != nil {
	panic(err)
}

// set a redirect programatically
r.RegexRedirects["^/from/(.*)"] = "/to/$1"

// append regex redirects from a CSV
if err := r.AppendRegexRedirects("/path/to/my/regex-redirects.csv"); er != nil {
	panic(err)
}

```

In a handler

```go
package foo

import(
	"net/http"
	"github.com/foomo/relaunchredirect"
)


type server struct {
	redirect *relaunchredirect.Redirect
}

func newServer() *server {
	return &server{
		redirect : relaunchredirect.NewRedirect(),
	}
}

func (s *server)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.Redirect.ShouldRedirect {
		s.Redirect.ServeHTTP(w, r)
		return
	}
	w.Write([]byte("do other stuff"))
}

```

