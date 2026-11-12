// release/ghreleases/ghreleases_test.go

package ghreleases

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// serve answers /repos/{owner}/{repo}/releases with body.
func serve(t *testing.T, status int, body string) *Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return New(Options{Owner: "smegg99", Repo: "SmeggTuner", BaseURL: srv.URL})
}

// GitHub returns drafts to any token with repo scope.
func TestDraftsAreNeverReturned(t *testing.T) {
	c := serve(t, 200, `[
		{"tag_name":"v2.0.0","draft":true,"body":"unpublished notes"},
		{"tag_name":"v1.0.0","draft":false,"body":"published"}
	]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d releases, want the draft dropped", len(got))
	}
	if got[0].Version != "v1.0.0" {
		t.Errorf("version = %q, want the published release", got[0].Version)
	}
	for _, r := range got {
		if strings.Contains(r.Body, "unpublished") {
			t.Fatal("a draft's notes were returned")
		}
	}
}

func TestPrereleasesAreReturnedFlagged(t *testing.T) {
	c := serve(t, 200, `[
		{"tag_name":"v2.0.0-beta","prerelease":true},
		{"tag_name":"v1.0.0","prerelease":false}
	]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d releases, want both", len(got))
	}
	if !got[0].Prerelease {
		t.Error("the beta is not flagged as a prerelease")
	}

	stable := Stable(got)
	if len(stable) != 1 || stable[0].Version != "v1.0.0" {
		t.Errorf("Stable = %+v, want only v1.0.0", stable)
	}
}

// Callers take the first element as latest.
func TestStableKeepsOrder(t *testing.T) {
	rs := []Release{
		{Version: "v3", Prerelease: true},
		{Version: "v2"},
		{Version: "v1"},
	}

	got := Stable(rs)

	if len(got) != 2 || got[0].Version != "v2" || got[1].Version != "v1" {
		t.Errorf("Stable = %+v, want v2 then v1", got)
	}
}

func TestAssetDigestIsCarriedThrough(t *testing.T) {
	c := serve(t, 200, `[{"tag_name":"v1.0.0","assets":[
		{"name":"a.AppImage","digest":"sha256:fc046371efa250e2875208341a786a35a01717d5eebec6903e199a9b8a3f3565"}
	]}]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := "sha256:fc046371efa250e2875208341a786a35a01717d5eebec6903e199a9b8a3f3565"
	if got[0].Assets[0].Digest != want {
		t.Errorf("digest = %q, want %q", got[0].Assets[0].Digest, want)
	}
}

// Assets predating GitHub's digests have none.
func TestMissingDigestStaysEmpty(t *testing.T) {
	c := serve(t, 200, `[{"tag_name":"v1.0.0","assets":[{"name":"old.zip"}]}]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].Assets[0].Digest != "" {
		t.Errorf("digest = %q, want empty", got[0].Assets[0].Digest)
	}
}

func TestSourceArchives(t *testing.T) {
	c := serve(t, 200, `[{
		"tag_name":"v1.2.0",
		"zipball_url":"https://api.github.com/repos/smegg99/SmeggTuner/zipball/v1.2.0",
		"tarball_url":"https://api.github.com/repos/smegg99/SmeggTuner/tarball/v1.2.0"
	}]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	src := got[0].Source
	if len(src) != 2 {
		t.Fatalf("got %d source archives, want zip and tar.gz", len(src))
	}

	if src[0].Name != "SmeggTuner-1.2.0.zip" || src[0].Kind != "zip" {
		t.Errorf("zip = %+v, want SmeggTuner-1.2.0.zip", src[0])
	}
	if src[1].Name != "SmeggTuner-1.2.0.tar.gz" || src[1].Kind != "tar.gz" {
		t.Errorf("tarball = %+v, want SmeggTuner-1.2.0.tar.gz", src[1])
	}
	for _, a := range src {
		if a.Platform != PlatformSource {
			t.Errorf("%s: platform = %q, want %q", a.Name, a.Platform, PlatformSource)
		}
		if a.Size != 0 || a.Digest != "" {
			t.Errorf("%s: size/digest = %d/%q, want neither, GitHub reports none", a.Name, a.Size, a.Digest)
		}
	}
}

// A release with no tag prefix must not lose a character to the "v" trim.
func TestSourceArchiveNamesWithoutAVPrefix(t *testing.T) {
	c := serve(t, 200, `[{"tag_name":"2026.07","zipball_url":"https://gh/z"}]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].Source[0].Name != "SmeggTuner-2026.07.zip" {
		t.Errorf("name = %q, want SmeggTuner-2026.07.zip", got[0].Source[0].Name)
	}
}

// Source archives can be disabled on a repo.
func TestNoSourceArchivesWhenGitHubOffersNone(t *testing.T) {
	c := serve(t, 200, `[{"tag_name":"v1.0.0"}]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got[0].Source) != 0 {
		t.Errorf("source = %+v, want none", got[0].Source)
	}
}

func TestAssetsAreClassified(t *testing.T) {
	c := serve(t, 200, `[{"tag_name":"v1.0.0","assets":[
		{"name":"SmeggTuner-1.0.0-x86_64.AppImage","size":42,"browser_download_url":"https://example.test/a"}
	]}]`)

	got, err := c.List(context.Background(), 30)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	a := got[0].Assets[0]
	if a.Platform != "linux" || a.Arch != "x86_64" || a.Kind != "appimage" {
		t.Errorf("asset = %+v, want linux/x86_64/appimage", a)
	}
	if a.DownloadURL != "https://example.test/a" || a.Size != 42 {
		t.Errorf("asset = %+v, want the url and size carried through", a)
	}
}

// No releases, or a private repo, so the caller can show an empty page.
func TestNotFoundIsItsOwnError(t *testing.T) {
	c := serve(t, http.StatusNotFound, `{"message":"Not Found"}`)

	_, err := c.List(context.Background(), 30)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUpstreamFailureIsAnError(t *testing.T) {
	c := serve(t, http.StatusInternalServerError, `{}`)

	if _, err := c.List(context.Background(), 30); err == nil {
		t.Error("err = nil, want a failure for a 500")
	}
}

func TestLimitIsSentAndClamped(t *testing.T) {
	cases := map[int]string{
		0:    "30",  // default
		10:   "10",  // as asked
		1000: "100", // GitHub's own ceiling
	}

	for limit, want := range cases {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query().Get("per_page")
				_, _ = w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			c := New(Options{Owner: "o", Repo: "r", BaseURL: srv.URL})
			if _, err := c.List(context.Background(), limit); err != nil {
				t.Fatalf("List: %v", err)
			}
			if got != want {
				t.Errorf("per_page = %q, want %q", got, want)
			}
		})
	}
}

func TestSendsTheHeadersGitHubAsksFor(t *testing.T) {
	var accept, agent, auth, version string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept, agent = r.Header.Get("Accept"), r.Header.Get("User-Agent")
		auth, version = r.Header.Get("Authorization"), r.Header.Get("X-GitHub-Api-Version")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := New(Options{Owner: "o", Repo: "r", BaseURL: srv.URL, Token: "t0ken", UserAgent: "smeggtuner-site"})
	if _, err := c.List(context.Background(), 30); err != nil {
		t.Fatalf("List: %v", err)
	}

	if accept != "application/vnd.github+json" {
		t.Errorf("Accept = %q", accept)
	}
	if agent != "smeggtuner-site" {
		t.Errorf("User-Agent = %q", agent)
	}
	if auth != "Bearer t0ken" {
		t.Errorf("Authorization = %q", auth)
	}
	if version == "" {
		t.Error("X-GitHub-Api-Version is not set")
	}
}

func TestNoTokenSendsNoAuthorization(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := New(Options{Owner: "o", Repo: "r", BaseURL: srv.URL})
	if _, err := c.List(context.Background(), 30); err != nil {
		t.Fatalf("List: %v", err)
	}
	if auth != "" {
		t.Errorf("Authorization = %q, want none", auth)
	}
}

// A slash in the owner or repo must not reach another path on the API.
func TestOwnerAndRepoAreEscaped(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := New(Options{Owner: "o/../..", Repo: "r", BaseURL: srv.URL})
	if _, err := c.List(context.Background(), 30); err != nil {
		t.Fatalf("List: %v", err)
	}
	if strings.Contains(path, "..") && !strings.Contains(path, url.PathEscape("o/../..")) {
		t.Errorf("path = %q, want the owner escaped rather than traversing", path)
	}
}
