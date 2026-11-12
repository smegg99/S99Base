// release/ghreleases/repo_test.go

package ghreleases

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepoReadsTheCounters(t *testing.T) {
	c := serve(t, 200, `{"stargazers_count":1234,"forks_count":56,"html_url":"https://github.com/smegg99/SmeggTuner"}`)

	got, err := c.Repo(context.Background())
	if err != nil {
		t.Fatalf("Repo: %v", err)
	}
	if got.Stars != 1234 {
		t.Errorf("stars = %d, want 1234", got.Stars)
	}
	if got.Forks != 56 {
		t.Errorf("forks = %d, want 56", got.Forks)
	}
	if got.HTMLURL != "https://github.com/smegg99/SmeggTuner" {
		t.Errorf("html_url = %q", got.HTMLURL)
	}
}

func TestRepoAsksForTheRepositoryItself(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"stargazers_count":1}`))
	}))
	t.Cleanup(srv.Close)

	c := New(Options{Owner: "smegg99", Repo: "SmeggTuner", BaseURL: srv.URL})
	if _, err := c.Repo(context.Background()); err != nil {
		t.Fatalf("Repo: %v", err)
	}
	if want := "/repos/smegg99/SmeggTuner"; path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestRepoNotFound(t *testing.T) {
	c := serve(t, 404, `{"message":"Not Found"}`)

	if _, err := c.Repo(context.Background()); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
