// release/ghreleases/repo.go

package ghreleases

import (
	"context"
	"fmt"
	"net/url"
)

// Repo holds the repository counters a download page displays.
type Repo struct {
	Stars   int    `json:"stars"`
	Forks   int    `json:"forks"`
	HTMLURL string `json:"html_url"`
}

type ghRepo struct {
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	HTMLURL         string `json:"html_url"`
}

// Repo returns ErrNotFound when the repository is private or does not exist.
func (c *Client) Repo(ctx context.Context) (Repo, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s",
		c.baseURL, url.PathEscape(c.owner), url.PathEscape(c.repo))

	var gh ghRepo
	if err := c.get(ctx, endpoint, &gh); err != nil {
		return Repo{}, err
	}

	return Repo{
		Stars:   gh.StargazersCount,
		Forks:   gh.ForksCount,
		HTMLURL: gh.HTMLURL,
	}, nil
}
