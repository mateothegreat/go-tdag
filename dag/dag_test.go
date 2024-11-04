package dag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MyCustomContext struct {
	Foo  string
	Date time.Time
}

func Test(t *testing.T) {
	dag := NewDag[MyCustomContext]()

	// Add nodes
	dag.AddNode("registration.register", func(t *testing.T) {
		t.Log("registration.register")
	})
	dag.AddNode("registration.confirm", func(t *testing.T) {
		t.Log("registration.confirm")
	})
	dag.AddNode("session.login", func(t *testing.T) {
		t.Log("session.login")
	})
	dag.AddNode("session.logout", func(t *testing.T) {
		t.Log("session.logout")
	})
	dag.AddNode("components.create", func(t *testing.T) {
		t.Log("components.create")
	})
	dag.AddNode("components.get", func(t *testing.T) {
		assert.Equal(t, "components.get", "components.get")
	})
	dag.AddNode("components.change.create", func(t *testing.T) {
		t.Log("components.change.create")
	})
	dag.AddNode("components.tags.create", func(t *testing.T) {
		t.Log("tags.create")
	})
	dag.AddNode("components.tags.delete", func(t *testing.T) {
		t.Log("tags.delete")
	})
	dag.AddNode("components.tags.get", func(t *testing.T) {
		t.Log("tags.get")
	})

	// Add edges with error handling for potential cycles
	if _, err := dag.AddEdge("registration.register", "registration.confirm"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("registration.confirm", "session.login"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("session.login", "components.create"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("components.create", "components.get"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("components.get", "components.change.create"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("components.change.create", "components.tags.create"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("components.tags.create", "components.tags.get"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("components.tags.get", "components.tags.delete"); err != nil {
		t.Fatal(err)
	}
	if _, err := dag.AddEdge("components.tags.delete", "session.logout"); err != nil {
		t.Fatal(err)
	}

	dag.RunTo("components.tags.create", t)
	// dag.RunTests(t)

	// Generate D2 output
	if err := dag.toD2("dag.d2"); err != nil {
		t.Fatal(err)
	}
}
