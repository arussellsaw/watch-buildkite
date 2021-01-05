package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/apoorvam/goterminal"
	"github.com/knq/emoji"
	"github.com/machinebox/graphql"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	client := graphql.NewClient("https://graphql.buildkite.com/v1")
	req := graphql.NewRequest(`query buildByBranch($branch: String!) {
		pipeline(slug: "sourcegraph/sourcegraph") {
			builds(branch: [$branch], first: 1) {
				edges {
					node {
						id
						commit
						createdAt
						jobs(first: 100) {
							edges {
								node {
									...on JobTypeCommand {
										id
										label
										command
										state
										exitStatus
									}
								}
							}
						}
					}
				}
			}
		}
	}`)
	req.Var("branch", strings.TrimSpace(buf.String()))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("BUILDKITE_TOKEN"))

	ctx := context.Background()
	res := Data{}
	err = client.Run(ctx, req, &res)
	if err != nil {
		panic(err)
	}
	writer := goterminal.New(os.Stdout)
	for {
		err = client.Run(ctx, req, &res)
		if err != nil {
			panic(err)
		}
		writer.Clear()
		for _, j := range res.Pipeline.Builds.Edges[0].Node.Jobs.Edges {
			if j.Node.Label == "" {
				continue
			}
			fmt.Fprintln(writer, statusEmoji(j.Node.State,j.Node.ExitStatus)," -", emoji.ReplaceAliases(j.Node.Label), j.Node.State)
			writer.Print()
		}
		time.Sleep(2 * time.Second)
	}
}

func statusEmoji(status ,exit string) string {
	switch status {
	case "FINISHED":
		if exit != "0" {
			return "❌"
		}
		return "✅"
	case "RUNNING":
		return "⏱ "
	case "CANCELING","CANCELED":
		return "❌"
	case "ASSIGNED":
		return "🔜"
	}
	return "❓"
}

type Response struct {
	Data Data `json:"data"`
}

type Data struct {
	Pipeline Pipeline `json:"pipeline"`
}

type Pipeline struct {
	Builds Builds `json:"builds"`
}

type Builds struct {
	Edges []BuildEdge `json:"edges"`
}

type BuildEdge struct {
	Node BuildNode `json:"node"`
}

type BuildNode struct {
	ID        string    `json:"id"`
	Commit    string    `json:"commit"`
	CreatedAt time.Time `json:"createdAt"`
	Jobs      Jobs      `json:"jobs"`
}

type Jobs struct {
	Edges []JobEdge `json:"edges"`
}

type JobNode struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Command string `json:"command"`
	State   string `json:"state"`
	ExitStatus string `json:"exitStatus"`
}
type JobEdge struct {
	Node JobNode `json:"node,omitempty"`
}
