package structs

import v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

type ResolverRequest struct {
	Payload    []byte `json:"payload"`
	Token      string `json:"token"`
	Provenance string `json:"provenance"`
}

type ResolverResponse struct {
	Payload      []byte            `json:"payload"`
	PipelineRuns []*v1.PipelineRun `json:"pipelineruns"`
}
