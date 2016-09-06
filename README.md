# relaunchredirect

Helps with redirects, when relaunching a website:

- redirects paths through a map lookup
- can force a domain
- can force TLS

## Usage

Instantiation and configuration:

```go

r := relaunchredirect.NewRedirect()

// force tls
r.ForceTLS = true

// force a host
r.ForceHost = "example.com"

// set a redirect programatically
r.Redirects["/from"] = "/to"

// load redirects from a CSV
csvLoadErr := r.LoadRedirects("/path/to/my/redirects.csv")
if csvLoadErr == nil {
	panic(csvLoadErr)
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

