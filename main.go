package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sigs.k8s.io/yaml"

	"github.com/piyush-garg/resolver-test/structs"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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
	fmt.Println("yooooooooooooooooo")

	handleSuccess(&w, request)
}

func handleSuccess(w *http.ResponseWriter, request structs.ResolverRequest) {
	writer := *w
	response := structs.ResolverResponse{}

	response.Payload = request.Payload
	pipelinerun := getPR()
	response.PipelineRuns = []*v1.PipelineRun{&pipelinerun}
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
