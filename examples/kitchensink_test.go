package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/mateothegreat/go-tdag"
	"github.com/stretchr/testify/assert"
)

func TestDag(t *testing.T) {
	dag := tdag.NewTDag(t)

	// This will be run first, before any node is executed.
	dag.Setup(func(ctx *tdag.TestContext) {
		ctx.Store.Set("setup", true)
		ctx.Store.Set("email", fmt.Sprintf("test-%d@example.com", time.Now().Unix()))
	})

	// This will be run last, after all nodes are executed.
	dag.TearDown(func(ctx *tdag.TestContext) {
		ctx.T.Log("tear down")
	})

	dag.BeforeEach(func(ctx *tdag.TestContext) {
		ctx.T.Log("before each")
	})

	dag.AfterEach(func(ctx *tdag.TestContext) {
		ctx.T.Log("after each")
	})

	dag.AddNode("registration.register", func(ctx *tdag.TestContext) {
		setup, err := ctx.Store.Get("setup")
		assert.NoError(ctx.T, err)
		assert.Equal(ctx.T, setup, true)
		ctx.T.Log("registration.register")
	})

	dag.AddNode("registration.confirm", func(ctx *tdag.TestContext) {
		ctx.T.Log("registration.confirm")
	})

	dag.AddNode("session.login", func(ctx *tdag.TestContext) {
		email, err := ctx.Store.Get("email")
		assert.NoError(ctx.T, err)
		ctx.T.Logf("session.login: %s", email)
	})

	dag.AddNode("session.logout", func(ctx *tdag.TestContext) {
		ctx.T.Log("session.logout")
	})

	dag.AddNode("components.create", func(ctx *tdag.TestContext) {
		ctx.T.Log("components.create")
	})

	dag.AddNode("components.get", func(ctx *tdag.TestContext) {
		ctx.T.Log("components.create")
	})

	dag.AddNode("components.change.create", func(ctx *tdag.TestContext) {
		ctx.T.Log("components.change.create")
	})

	dag.AddNode("components.tags.create", func(ctx *tdag.TestContext) {
		ctx.T.Log("tags.create")
	})

	dag.AddNode("components.tags.delete", func(ctx *tdag.TestContext) {
		ctx.T.Log("tags.delete")
	})

	dag.AddNode("components.tags.get", func(ctx *tdag.TestContext) {
		ctx.T.Log("tags.get")
	})

	vertexes := [][]string{
		{"registration.register", "registration.confirm"},
		{"registration.confirm", "session.login"},
		{"session.login", "components.create"},
		{"components.create", "components.get"},
		{"components.get", "components.change.create"},
		{"components.change.create", "components.tags.create"},
		{"components.tags.create", "components.tags.get"},
		{"components.tags.get", "components.tags.delete"},
		{"components.tags.delete", "session.logout"},
	}

	for _, vertex := range vertexes {
		if _, err := dag.AddEdge(vertex[0], vertex[1]); err != nil {
			t.Fatal(err)
		}
	}

	dag.RunTo("components.tags.create", t)
	// dag.RunTests(t)

	// Generate D2 output
	if err := dag.ToD2("dag.d2"); err != nil {
		t.Fatal(err)
	}
}
