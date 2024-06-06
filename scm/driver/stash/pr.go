// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stash

import (
	"context"
	"fmt"
	"time"

	"github.com/drone/go-scm/scm"
)

type pullService struct {
	client *wrapper
}

func (s *pullService) Find(ctx context.Context, repo string, number int) (*scm.PullRequest, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", namespace, name, number)
	out := new(pr)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertPullRequest(out), res, err
}

func (s *pullService) FindComment(ctx context.Context, repo string, number int, id int) (*scm.Comment, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d", namespace, name, number, id)
	out := new(pullRequestComment)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertPullRequestComment(*out, 0, nil), res, err
}

func (s *pullService) List(ctx context.Context, repo string, opts scm.PullRequestListOptions) ([]*scm.PullRequest, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests", namespace, name)
	out := new(prs)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	if !out.pagination.LastPage.Bool {
		res.Page.First = 1
		res.Page.Next = opts.Page + 1
	}
	return convertPullRequests(out), res, err
}

func (s *pullService) ListChanges(ctx context.Context, repo string, number int, opts scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/changes", namespace, name, number)
	out := new(diffstats)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	if !out.pagination.LastPage.Bool {
		res.Page.First = 1
		res.Page.Next = opts.Page + 1
	}
	return convertDiffstats(out), res, err
}

func (s *pullService) ListComments(ctx context.Context, repo string, id int, opts scm.ListOptions) ([]*scm.Comment, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/activities", namespace, name, id)
	out := new(Activities)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertPullRequestCommentsList(out.Values), res, err
}

func (s *pullService) ListCommits(ctx context.Context, repo string, number int, opts scm.ListOptions) ([]*scm.Commit, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/commits?%s", namespace, name, number, encodeListOptionsV2(opts))
	out := new(commits)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	if !out.pagination.LastPage.Bool {
		res.Page.First = 1
		res.Page.Next = opts.Page + 1
	}
	return convertCommitList(out), res, err
}

func (s *pullService) Merge(ctx context.Context, repo string, number int) (*scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge", namespace, name, number)
	res, err := s.client.do(ctx, "POST", path, nil, nil)
	return res, err
}

func (s *pullService) Close(ctx context.Context, repo string, number int) (*scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/decline", namespace, name, number)
	res, err := s.client.do(ctx, "POST", path, nil, nil)
	return res, err
}

func (s *pullService) Create(ctx context.Context, repo string, input *scm.PullRequestInput) (*scm.PullRequest, *scm.Response, error) {
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests", namespace, name)
	in := new(prInput)
	in.Title = input.Title
	in.Description = input.Body
	in.FromRef.Repository.Project.Key = namespace
	in.FromRef.Repository.Slug = name
	in.FromRef.ID = scm.ExpandRef(input.Source, "refs/heads")
	in.ToRef.Repository.Project.Key = namespace
	in.ToRef.Repository.Slug = name
	in.ToRef.ID = scm.ExpandRef(input.Target, "refs/heads")
	out := new(pr)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertPullRequest(out), res, err
}

func (s *pullService) CreateComment(ctx context.Context, repo string, number int, in *scm.CommentInput) (*scm.Comment, *scm.Response, error) {
	input := pullRequestCommentInput{Text: in.Body}
	namespace, name := scm.Split(repo)
	path := fmt.Sprintf("rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments", namespace, name, number)
	out := new(pullRequestComment)
	res, err := s.client.do(ctx, "POST", path, &input, out)
	return convertPullRequestComment(*out, 0, nil), res, err
}

func (s *pullService) DeleteComment(context.Context, string, int, int) (*scm.Response, error) {
	// TODO(bradrydzewski) the challenge with deleting comments is that we need to specify
	// the comment version number. The proposal is to use 0 as the initial version number,
	// and then to use expectedVersion on error and re-attempt the API call.

	// DELETE /rest/api/1.0/projects/PRJ/repos/my-repo/pull-requests/1/comments/1?version=0
	return nil, scm.ErrNotSupported
}

type Author struct {
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	ID           int    `json:"id"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	Slug         string `json:"slug"`
	Type         string `json:"type"`
	Links        struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

type pr struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Open        bool   `json:"open"`
	Closed      bool   `json:"closed"`
	CreatedDate int64  `json:"createdDate"`
	UpdatedDate int64  `json:"updatedDate"`
	FromRef     struct {
		ID           string     `json:"id"`
		DisplayID    string     `json:"displayId"`
		LatestCommit string     `json:"latestCommit"`
		Repository   repository `json:"repository"`
	} `json:"fromRef"`
	ToRef struct {
		ID           string     `json:"id"`
		DisplayID    string     `json:"displayId"`
		LatestCommit string     `json:"latestCommit"`
		Repository   repository `json:"repository"`
	} `json:"toRef"`
	Locked bool `json:"locked"`
	Author struct {
		User     Author `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
		Status   string `json:"status"`
	} `json:"author"`
	Reviewers    []interface{} `json:"reviewers"`
	Participants []interface{} `json:"participants"`
	Links        struct {
		Self []link `json:"self"`
	} `json:"links"`
	Properties struct {
		MergeCommit struct {
			ID        string `json:"id"`
			DisplayID string `json:"displayId"`
		} `json:"mergeCommit"`
	} `json:"properties"`
}

type prs struct {
	pagination
	Values []*pr `json:"values"`
}

type prInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	FromRef     struct {
		ID         string `json:"id"`
		Repository struct {
			Slug    string `json:"slug"`
			Project struct {
				Key string `json:"key"`
			} `json:"project"`
		} `json:"repository"`
	} `json:"fromRef"`
	ToRef struct {
		ID         string `json:"id"`
		Repository struct {
			Slug    string `json:"slug"`
			Project struct {
				Key string `json:"key"`
			} `json:"project"`
		} `json:"repository"`
	} `json:"toRef"`
}

func convertPullRequests(from *prs) []*scm.PullRequest {
	to := []*scm.PullRequest{}
	for _, v := range from.Values {
		to = append(to, convertPullRequest(v))
	}
	return to
}

func convertPullRequest(from *pr) *scm.PullRequest {
	fork := scm.Join(
		from.FromRef.Repository.Project.Key,
		from.FromRef.Repository.Slug,
	)
	return &scm.PullRequest{
		Number:  from.ID,
		Title:   from.Title,
		Body:    from.Description,
		Sha:     from.FromRef.LatestCommit,
		Merge:   from.Properties.MergeCommit.ID,
		Ref:     fmt.Sprintf("refs/pull-requests/%d/from", from.ID),
		Source:  from.FromRef.DisplayID,
		Target:  from.ToRef.DisplayID,
		Fork:    fork,
		Link:    extractSelfLink(from.Links.Self),
		Closed:  from.Closed,
		Merged:  from.State == "MERGED",
		Created: time.Unix(from.CreatedDate/1000, 0),
		Updated: time.Unix(from.UpdatedDate/1000, 0),
		Head: scm.Reference{
			Name: from.FromRef.DisplayID,
			Path: from.FromRef.ID,
			Sha:  from.FromRef.LatestCommit,
		},
		Base: scm.Reference{
			Name: from.ToRef.DisplayID,
			Path: from.ToRef.ID,
			Sha:  from.ToRef.LatestCommit,
		},
		Author: scm.User{
			Login:  from.Author.User.Slug,
			Name:   from.Author.User.DisplayName,
			Email:  from.Author.User.EmailAddress,
			Avatar: avatarLink(from.Author.User.EmailAddress),
		},
	}
}

type CommentAnchor struct {
	FromHash string `json:"fromHash"`
	ToHash   string `json:"toHash"`
	Line     int    `json:"line"`
	LineType string `json:"lineType"`
	FileType string `json:"fileType"`
	Path     string `json:"path"`
	DiffType string `json:"diffType"`
	Orphaned bool   `json:"orphaned"`
}

type PRCommentActivity struct {
	ID            int                `json:"id"`
	CreatedDate   int64              `json:"createdDate"`
	User          Author             `json:"user"`
	Action        string             `json:"action"`
	CommentAction string             `json:"commentAction"`
	Comment       pullRequestComment `json:"comment"`
	CommentAnchor `json:"commentAnchor"`
	Diff          struct {
		Source      interface{}   `json:"source"`
		Destination interface{}   `json:"destination"`
		Hunks       []interface{} `json:"hunks"`
		Truncated   bool          `json:"truncated"`
		Properties  interface{}   `json:"properties"`
	}
}

type Activities struct {
	pagination
	Values []interface{} `json:"values"`
}

type pullRequestComment struct {
	Properties struct {
		RepositoryID int `json:"repositoryId"`
	} `json:"properties"`
	ID                  int    `json:"id"`
	Version             int    `json:"version"`
	Text                string `json:"text"`
	Author              `json:"author"`
	CreatedDate         int64                `json:"createdDate"`
	UpdatedDate         int64                `json:"updatedDate"`
	Comments            []pullRequestComment `json:"comments"`
	Tasks               []interface{}        `json:"tasks"`
	PermittedOperations struct {
		Editable  bool `json:"editable"`
		Deletable bool `json:"deletable"`
	} `json:"permittedOperations"`
}

type pullRequestCommentInput struct {
	Text string `json:"text"`
}

func filterOutComments(from []interface{}) []PRCommentActivity {
	comments := []PRCommentActivity{}
	for _, activity := range from {
		// Type assertion to check if the activity is a PRCommentActivity
		if prComment, ok := activity.(PRCommentActivity); ok {
			if prComment.Action == "COMMENTED" {
				comments = append(comments, prComment)
			}
		}
	}
	return comments
}

func convertPullRequestCommentsList(from []interface{}) []*scm.Comment {
	comments := filterOutComments(from)
	to := []*scm.Comment{}
	for _, c := range comments {
		comment := c.Comment
		to = append(to, convertPullRequestComment(comment, 0, &c.CommentAnchor))
		childComments := comment.Comments
		for len(childComments) > 0 {
			if len(childComments) == 0 {
				break
			}
			childComment := childComments[0]
			to = append(to, convertPullRequestComment(childComment, comment.ID, nil))
		}
	}
	return to
}

func convertPullRequestComment(from pullRequestComment, parentId int, anchor *CommentAnchor) *scm.Comment {
	var metadata interface{}
	commentType := "comment"
	metadata = scm.GeneralCommentMetadata{}
	if anchor != nil && (*anchor).Path != "" {
		commentAnchor := *anchor
		commentType = "code-comment"
		metadata = scm.CodeCommentMetadata{
			Path:         commentAnchor.Path,
			Line:         commentAnchor.Line,
			LineSpan:     1,
			SourceSha:    commentAnchor.FromHash,
			MergeBaseSha: commentAnchor.ToHash,
		}
	}
	if parentId != 0 {
		commentType = "reply-comment"
		metadata = scm.ReplyCommentMetadata{
			ParentID: parentId,
		}
	}
	return &scm.Comment{
		ID:      from.ID,
		Body:    from.Text,
		Created: time.Unix(from.CreatedDate/1000, 0),
		Updated: time.Unix(from.UpdatedDate/1000, 0),
		Author: scm.User{
			Login:  from.Author.Slug,
			Name:   from.Author.DisplayName,
			Email:  from.Author.EmailAddress,
			Avatar: avatarLink(from.Author.EmailAddress),
		},
		Type:     commentType,
		Metadata: metadata,
	}
}
