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

func expect(rec *httptest.ResponseRecorder, expectedStatus int, comment string, t *testing.T) {
	if rec.Code != expectedStatus {
		t.Fatal("unexpected status:", rec.Code, "!=", expectedStatus, "comment:", comment, ", after requesting", rec.Header().Get("debug"))
	}
}

func TestForceHost(t *testing.T) {
	r := NewRedirect()
	expect(runTest(r, "http://foo.com/"), http.StatusOK, "no redirect", t)
	r.ForceHost = "bar.com"
	expect(runTest(r, "http://foo.com/"), http.StatusMovedPermanently, "redirect", t)
}

func TestRedirects(t *testing.T) {
	r := NewRedirect()
	r.Redirects["/foo"] = "/"
	r.Redirects["/a"] = "/"
	expect(runTest(r, "http://foo.com/foo"), http.StatusMovedPermanently, "redirect /foo", t)
	expect(runTest(r, "http://foo.com/a"), http.StatusMovedPermanently, "redirect /a", t)
	expect(runTest(r, "http://foo.com/ok"), http.StatusOK, "200", t)
}

func TestLoadRedirects(t *testing.T) {
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

func TestForceTLS(t *testing.T) {
	r := NewRedirect()
	expect(runTest(r, "http://foo.com/ok"), http.StatusOK, "200", t)
	r.ForceTLS = true
	expect(runTest(r, "http://foo.com/ok"), http.StatusMovedPermanently, "tls redirect", t)
}
