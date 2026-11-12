// release/ghreleases/ghreleases.go

// Package ghreleases reads one GitHub repository's releases and classifies their assets.
package ghreleases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://api.github.com"
	defaultUserAgent = "ghreleases"
	defaultLimit     = 30

	// maxLimit is GitHub's own per_page ceiling.
	maxLimit = 100

	maxBody = 4 << 20 // 4 MB
)

// Options configures a Client; only Owner and Repo are required.
type Options struct {
	Owner string
	Repo  string

	// BaseURL defaults to GitHub's public API.
	BaseURL string

	// Token is optional and only raises the rate limit. Read it from the environment.
	Token string

	// UserAgent should name the application, as GitHub asks.
	UserAgent string

	// HTTPClient defaults to a client with a 10s timeout.
	HTTPClient *http.Client
}

type Client struct {
	owner, repo string
	baseURL     string
	token       string
	userAgent   string
	http        *http.Client
}

func New(opts Options) *Client {
	c := &Client{
		owner:     opts.Owner,
		repo:      opts.Repo,
		baseURL:   strings.TrimRight(opts.BaseURL, "/"),
		token:     opts.Token,
		userAgent: opts.UserAgent,
		http:      opts.HTTPClient,
	}
	if c.baseURL == "" {
		c.baseURL = defaultBaseURL
	}
	if c.userAgent == "" {
		c.userAgent = defaultUserAgent
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 10 * time.Second}
	}
	return c
}

// Release is one published release.
type Release struct {
	Version     string  `json:"version"`
	Name        string  `json:"name"`
	PublishedAt string  `json:"published_at"`
	HTMLURL     string  `json:"html_url"`
	Body        string  `json:"body"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets"`

	// Source is the tag zip and tarball GitHub generates; they have no digest or size.
	Source []Asset `json:"source"`
}

// Asset is one downloadable file, classified from its filename.
type Asset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
	Platform    string `json:"platform"` // linux | windows | macos | android | web | source | other
	Arch        string `json:"arch"`     // x86_64 | arm64 | universal | ""
	Kind        string `json:"kind"`     // appimage | deb | rpm | pkg.tar.zst | tar.gz | tar.zst | installer | exe | msix | dmg | zip | apk | aab | binary | ext

	// Digest is "algo:hex", empty for older assets and Source. It catches a truncated
	// download or a tampering mirror; it is not a signature.
	Digest string `json:"digest"`
}

// PlatformSource marks the generated source archives; PlatformOther an asset for no platform.
const (
	PlatformSource = "source"
	PlatformOther  = "other"
)

type ghRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	ZipballURL  string `json:"zipball_url"`
	TarballURL  string `json:"tarball_url"`
	Assets      []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

// List returns up to limit releases newest first (default 30, max 100), dropping drafts and flagging prereleases.
func (c *Client) List(ctx context.Context, limit int) ([]Release, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d",
		c.baseURL, url.PathEscape(c.owner), url.PathEscape(c.repo), limit)

	var gh []ghRelease
	if err := c.get(ctx, endpoint, &gh); err != nil {
		return nil, err
	}

	out := make([]Release, 0, len(gh))
	for i := range gh {
		if gh[i].Draft {
			continue
		}
		out = append(out, c.convert(&gh[i]))
	}
	return out, nil
}

// ErrNotFound means the repository has no releases, is private, or does not exist.
var ErrNotFound = fmt.Errorf("ghreleases: not found")

func (c *Client) get(ctx context.Context, endpoint string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ghreleases: github status %d", resp.StatusCode)
	}

	return json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(into)
}

func (c *Client) convert(gh *ghRelease) Release {
	rel := Release{
		Version:     gh.TagName,
		Name:        gh.Name,
		PublishedAt: gh.PublishedAt,
		HTMLURL:     gh.HTMLURL,
		Body:        gh.Body,
		Prerelease:  gh.Prerelease,
		Assets:      make([]Asset, 0, len(gh.Assets)),
		Source:      c.sourceArchives(gh),
	}
	for _, a := range gh.Assets {
		platform, arch, kind := classify(a.Name)
		rel.Assets = append(rel.Assets, Asset{
			Name:        a.Name,
			Size:        a.Size,
			DownloadURL: a.BrowserDownloadURL,
			Platform:    platform,
			Arch:        arch,
			Kind:        kind,
			Digest:      a.Digest,
		})
	}
	return rel
}

// sourceArchives names the tag archives repo-tag.zip and .tar.gz, never nil so JSON gets [].
func (c *Client) sourceArchives(gh *ghRelease) []Asset {
	stem := c.repo + "-" + strings.TrimPrefix(gh.TagName, "v")

	out := make([]Asset, 0, 2)
	if gh.ZipballURL != "" {
		out = append(out, Asset{
			Name:        stem + ".zip",
			DownloadURL: gh.ZipballURL,
			Platform:    PlatformSource,
			Kind:        "zip",
		})
	}
	if gh.TarballURL != "" {
		out = append(out, Asset{
			Name:        stem + ".tar.gz",
			DownloadURL: gh.TarballURL,
			Platform:    PlatformSource,
			Kind:        "tar.gz",
		})
	}
	return out
}

// Stable drops prereleases and keeps order, so the first element is the newest stable.
func Stable(rs []Release) []Release {
	out := make([]Release, 0, len(rs))
	for _, r := range rs {
		if !r.Prerelease {
			out = append(out, r)
		}
	}
	return out
}
