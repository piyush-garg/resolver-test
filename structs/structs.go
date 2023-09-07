package structs

import v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

type Data struct {
	EventType  string `json:"eventType,omitempty"`
	BaseBranch string `json:"baseBranch,omitempty"`
	HeadBranch string `json:"headBranch,omitempty"`
	BaseURL    string `json:"baseURL,omitempty"`
	HeadURL    string `json:"headURL,omitempty"`
	SHA        string `json:"sha,omitempty"`

	// Github
	GithubOrganization   string `json:"githubOrganization,omitempty"`
	GithubRepository     string `json:"githubRepository,omitempty"`
	GithubInstallationID int64  `json:"githubInstallationID,omitempty"`

	// GHE
	GHEURL string `json:"gheURL,omitempty"`

	// Bitbucket Cloud
	BitBucketAccountID string `json:"bitBucketAccountID,omitempty"`

	// Bitbucket Server
	BitBucketCloneURL string `json:"bitBucketCloneURL,omitempty"`

	// Gitlab
	GitlabSourceProjectID int `json:"gitlabSourceProjectID,omitempty"`
	GitlabTargetProjectID int `json:"gitlabTargetProjectID,omitempty"`
}

type ResolverRequest struct {
	Data  string `json:"data"`
	Token string `json:"token"`
}

type ResolverResponse struct {
	PipelineRuns []*v1.PipelineRun `json:"pipelineruns"`
}
