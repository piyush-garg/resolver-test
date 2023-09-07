package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/piyush-garg/resolver-test/structs"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"sigs.k8s.io/yaml"
)

func main() {
	log.Println("Attempting to start HTTP Server.")
	mux := http.NewServeMux()
	mux.HandleFunc("/resolve", handleRequest)
	var err = http.ListenAndServe(":8000", mux)
	if err != nil {
		log.Panicln("Server failed starting. Error: %s", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)
	byteData, err := io.ReadAll(r.Body)
	if err != nil {
		handleError(&w, 500, "Internal Server Error", "Error reading data from body", err)
		return
	}

	request := structs.ResolverRequest{}
	err = json.Unmarshal(byteData, &request)
	if err != nil {
		handleError(&w, 500, "Internal Server Error", "Error unmarhsalling JSON", err)
		return
	}

	handleSuccess(&w, request)
}

func handleSuccess(w *http.ResponseWriter, request structs.ResolverRequest) {
	writer := *w
	response := structs.ResolverResponse{}

	payloadData := structs.Data{}
	if err := decodeFromBase64(&payloadData, request.Data); err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error decoding data string", err)
		return
	}

	pipelinerun, err := clone(payloadData, request.Token)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error cloning", err)
		return
	}
	response.PipelineRuns = pipelinerun
	responseMarshalled, err := json.Marshal(response)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error marshalling response JSON", err)
		return
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(responseMarshalled)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error writing response JSON", err)
		return
	}
}

func handleError(w *http.ResponseWriter, code int, responseText string, logMessage string, err error) {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}

	log.Println(logMessage, errorMessage)
	writer := *w
	writer.WriteHeader(code)
	writer.Write([]byte(responseText))
}

func getPR() v1.PipelineRun {
	var p v1.PipelineRun
	err := yaml.Unmarshal([]byte(pr), &p)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	fmt.Println(p)
	return p
}

var pr = `
---
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: article-no-operation-test
  annotations:
    # The event we are targeting as seen from the webhook payload
    # this can be an array too, i.e: [pull_request, push]
    # pipelinesascode.tekton.dev/on-event: "[pull_request, push]"
    pipelinesascode.tekton.dev/on-event: "[pull_request, push]"
    # pipelinesascode.tekton.dev/on-event: "[incoming]"

    # The branch or tag we are targeting (ie: main, refs/tags/*)
    pipelinesascode.tekton.dev/on-target-branch: "[main]"

    # Fetch the git-clone task from hub, we are able to reference later on it
    # with taskRef and it will automatically be embedded into our pipeline.
    # pipelinesascode.tekton.dev/task: "git-clone"

    # Use maven task from hub
    #
    # pipelinesascode.tekton.dev/task-1: "maven"

    # You can add more tasks by increasing the suffix number, you can specify them as array to have multiple of them.
    # browse the tasks you want to include from hub on https://hub.tekton.dev/
    #
    # pipelinesascode.tekton.dev/task-2: "[curl, buildah]"

    # How many runs we want to keep.
    pipelinesascode.tekton.dev/max-keep-runs: "5"
      # pipelinesascode.tekton.dev/on-cel-expression: |
    # event == "push" && target_branch == "main" && "frontend/***".pathChanged()
spec:
  pipelineSpec:
    tasks:
      # Customize this task if you like, or just do a taskRef
      # to one of the hub task.
      - name: noop-task
        taskSpec:
          steps:
            - name: noop-task
              image: registry.access.redhat.com/ubi9/ubi-micro
              script: |
                echo "Hello"
                # sleep 30
                exit 0
`

func decodeFromBase64(v interface{}, enc string) error {
	return json.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(enc))).Decode(v)
}

func clone(payloadData structs.Data, token string) ([]*v1.PipelineRun, error) {
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s/%s", payloadData.GithubOrganization, payloadData.GithubRepository),
		Auth: &githttp.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: token,
		},
		ReferenceName: plumbing.NewBranchReferenceName(payloadData.HeadBranch),
		Progress:      os.Stdout,
	})
	if err != nil {
		return nil, err
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}

	// ... retrieving the commit object
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	tektontree, err := tree.Tree(".tekton")
	if err != nil {
		return nil, err
	}

	var prs []*v1.PipelineRun
	tektontree.Files().ForEach(func(f *object.File) error {
		if strings.HasSuffix(f.Name, "yaml") {
			filecontent, err := f.Contents()
			if err != nil {
				return err
			}
			var p v1.PipelineRun
			err = yaml.Unmarshal([]byte(filecontent), &p)
			if err != nil {
				return err
			}
			prs = append(prs, &p)
		}
		return nil
	})
	for _, pr := range prs {
		pr.Name = "test-resolver" + pr.Name
	}
	return prs, nil
}
