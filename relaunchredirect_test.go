package relaunchredirect

import (
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"testing"
)

const reply200 = "Hello"

func GetCurrentDir() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

func runTest(r *Redirect, url string) (rec *httptest.ResponseRecorder) {
	rec = httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)
	func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("debug", req.URL.String())
		if r.ShouldRedirect(req) {
			r.ServeHTTP(w, req)
			return
		}
		w.Write([]byte(reply200))
	}(rec, req)
	return rec
}

func expect(rec *httptest.ResponseRecorder, expectedStatus int, expectedValue string, comment string, t *testing.T) {
	loc, _ := rec.Result().Location()
	if rec.Code != expectedStatus {
		t.Log(loc.String())
		t.Fatal("unexpected status:", rec.Code, "!=", expectedStatus, "comment:", comment, ", after requesting", rec.Header().Get("debug"))
	} else if expectedStatus == http.StatusMovedPermanently &&  expectedValue != loc.String() {
		t.Fatal("unexpected redirect url:", loc.String(), "!=", expectedValue, "comment:", comment, ", after requesting", rec.Header().Get("debug"))
	}
}

func TestForceHost(t *testing.T) {
	r := NewRedirect()
	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "no redirect", t)
	r.ForceHost = "bar.com"
	expect(runTest(r, "http://foo.com/"), http.StatusMovedPermanently,"http://bar.com/", "redirect", t)
}

func TestRedirects(t *testing.T) {
	r := NewRedirect()
	r.Redirects["/foo"] = "/"
	r.Redirects["/a"] = "/"
	expect(runTest(r, "http://foo.com/foo"), http.StatusMovedPermanently, "http://foo.com/", "redirect /foo", t)
	expect(runTest(r, "http://foo.com/a"), http.StatusMovedPermanently, "http://foo.com/", "redirect /a", t)
	expect(runTest(r, "http://foo.com/ok"), http.StatusOK,  "","200", t)
}

func TestRegexRedirects(t *testing.T) {
	r := NewRedirect()
	r.RegexRedirects["^/foo/(.*)"] = "/$1"
	expect(runTest(r, "http://foo.com/foo/bar"), http.StatusMovedPermanently, "http://foo.com/bar", "redirect /bar", t)
	expect(runTest(r, "http://foo.com/ok"), http.StatusOK, "", "200", t)
}

func TestAppendRedirects(t *testing.T) {
	r := NewRedirect()
	err := r.AppendRedirects("c://file/no/have")
	if err == nil {
		t.Fatal("that should have failed")
	}
	err = r.AppendRedirects(path.Join(GetCurrentDir(), "redirects_example.csv"))
	if err != nil {
		t.Fatal("csv loading should have worked")
	}
}

func TestAppendRegexRedirects(t *testing.T) {
	r := NewRedirect()
	err := r.AppendRegexRedirects("c://file/no/have")
	if err == nil {
		t.Fatal("that should have failed")
	}
	err = r.AppendRegexRedirects(path.Join(GetCurrentDir(), "regex_redirects_example.csv"))
	if err != nil {
		t.Fatal("csv loading should have worked")
	}
}

func TestForceTLS(t *testing.T) {
	r := NewRedirect()
	expect(runTest(r, "http://foo.com/ok"), http.StatusOK, "", "200", t)
	r.ForceTLS = true
	expect(runTest(r, "http://foo.com/ok"), http.StatusMovedPermanently, "https://foo.com/ok", "tls redirect", t)
}

func TestForceLowerCase(t *testing.T) {
	r := NewRedirect()
	expect(runTest(r, "http://foo.com/Foo"), http.StatusOK, "", "200", t)

	r.ForceLowerCase = true
	expect(runTest(r, "http://foo.com/foo"), http.StatusOK, "", "lower case redirect", t)
	expect(runTest(r, "http://foo.com/Foo"), http.StatusMovedPermanently, "http://foo.com/foo", "lower case redirect", t)
	expect(runTest(r, "http://foo.com/ignore/Foo"), http.StatusMovedPermanently, "http://foo.com/ignore/foo", "lower case redirect", t)

	r.ForceLowerCaseIgnore = "^/ignore/(.*)"

	expect(runTest(r, "http://foo.com/foo"), http.StatusOK, "", "lower case redirect", t)
	expect(runTest(r, "http://foo.com/Foo"), http.StatusMovedPermanently, "http://foo.com/foo", "lower case redirect", t)
	expect(runTest(r, "http://foo.com/ignore"), http.StatusOK, "", "lower case redirect", t)
	expect(runTest(r, "http://foo.com/ignore/Foo"), http.StatusOK, "", "lower case redirect", t)
	expect(runTest(r, "http://foo.com/IGNORE/Foo"), http.StatusMovedPermanently, "http://foo.com/ignore/foo", "lower case redirect", t)
}

func TestForceTrailingSlashes(t *testing.T) {
	r := NewRedirect()
	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://foo.com/foo"), http.StatusOK, "", "200", t)

	r.ForceTrailingSlash = true

	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://foo.com/foo"), http.StatusMovedPermanently, "http://foo.com/foo/", "trailing slashes redirect", t)
	expect(runTest(r, "http://foo.com/ignore/foo"), http.StatusMovedPermanently, "http://foo.com/ignore/foo/", "trailing slashes redirect", t)

	r.ForceTrailingSlashIgnore = "^/ignore/(.*)"

	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://foo.com/foo"), http.StatusMovedPermanently, "http://foo.com/foo/", "trailing slashes redirect", t)
	expect(runTest(r, "http://foo.com/ignore/foo"), http.StatusOK, "", "trailing slashes redirect", t)
}

func TestForceNoTrailingSlashes(t *testing.T) {
	r := NewRedirect()

	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://foo.com/foo/"), http.StatusOK, "", "200", t)

	r.ForceNoTrailingSlash = true

	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://foo.com/foo/"), http.StatusMovedPermanently, "http://foo.com/foo", "no trailing slashes redirect", t)
	expect(runTest(r, "http://foo.com/ignore/foo/"), http.StatusMovedPermanently, "http://foo.com/ignore/foo", "trailing slashes redirect", t)

	r.ForceNoTrailingSlashIgnore = "^/ignore/(.*)"

	expect(runTest(r, "http://foo.com/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://foo.com/foo/"), http.StatusMovedPermanently, "http://foo.com/foo", "no trailing slashes redirect", t)
	expect(runTest(r, "http://foo.com/ignore/"), http.StatusOK, "http://foo.com/ignore/", "trailing slashes redirect", t)
	expect(runTest(r, "http://foo.com/ignore/foo/"), http.StatusOK, "http://foo.com/ignore/foo/", "trailing slashes redirect", t)
}

func TestComplex(t *testing.T) {
	r := NewRedirect()

	r.ForceHost = "www.foo.com"
	r.ForceLowerCase = true
	r.ForceLowerCaseIgnore = "^/ignore/(.*)"
	r.ForceNoTrailingSlash = true
	r.ForceNoTrailingSlashIgnore = "^/ignore/(.*)"

	expect(runTest(r, "http://foo.com/"), http.StatusMovedPermanently, "http://www.foo.com/", "200", t)
	expect(runTest(r, "http://foo.com/foo/"), http.StatusMovedPermanently, "http://www.foo.com/foo", "200", t)
	expect(runTest(r, "http://www.foo.com/Foo"), http.StatusMovedPermanently, "http://www.foo.com/foo", "200", t)
	expect(runTest(r, "http://www.foo.com/ignore/"), http.StatusOK, "", "200", t)
	expect(runTest(r, "http://www.foo.com/ignore/Foo"), http.StatusOK, "", "200", t)
}