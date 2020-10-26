package query_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/db/query"
)

func ExampleEval() {
	q := query.And(
		query.Or("foo", false),
		query.Not("bar"),
		true)

	rq, err := query.Eval(q, func(q query.Q) (*sqlf.Query, error) {
		p, ok := q.(string)
		if !ok {
			return nil, errors.Errorf("unexpected token in query: %q", q)
		}
		return sqlf.Sprintf("name LIKE %s", "%"+p+"%"), nil
	})
	if err != nil {
		log.Fatal("unexpected error", err)
	}

	fmt.Println("Expression:", query.Print(q))
	fmt.Println("SQL Query: ", rq.Query(sqlf.PostgresBindVar))
	fmt.Println("SQL Args:  ", rq.Args())
	// Output:
	// Expression: (("foo" OR FALSE) AND NOT("bar") AND TRUE)
	// SQL Query:  ((name LIKE $1 OR FALSE) AND NOT(name LIKE $2) AND TRUE)
	// SQL Args:   [%foo% %bar%]
}

func ExampleEval_types() {
	type equal struct{ Value string }
	type prefix struct{ Value string }
	q := query.And(
		&prefix{"github.com/gorilla/"},
		query.Not(&equal{"github.com/gorilla/mux"}))

	rq, err := query.Eval(q, func(q query.Q) (*sqlf.Query, error) {
		switch c := q.(type) {
		case *prefix:
			return sqlf.Sprintf("name LIKE %s", c.Value+"%"), nil
		case *equal:
			return sqlf.Sprintf("name = %s", c.Value), nil
		default:
			return nil, errors.Errorf("unexpected token in query: %q", q)
		}
	})
	if err != nil {
		log.Fatal("unexpected error", err)
	}

	fmt.Println("Expression:", query.Print(q))
	fmt.Println("SQL Query: ", rq.Query(sqlf.PostgresBindVar))
	fmt.Println("SQL Args:  ", rq.Args())
	// Output:
	// Expression: (&query_test.prefix{Value:"github.com/gorilla/"} AND NOT(&query_test.equal{Value:"github.com/gorilla/mux"}))
	// SQL Query:  (name LIKE $1 AND NOT(name = $2))
	// SQL Args:   [github.com/gorilla/% github.com/gorilla/mux]
}

func TestEval_cornercase(t *testing.T) {
	cases := []struct {
		q    query.Q
		want string
	}{{
		q:    query.And(),
		want: "TRUE",
	}, {
		q:    query.And("test"),
		want: `("test")`,
	}, {
		q:    query.Or(),
		want: "FALSE",
	}, {
		q:    query.Or("test"),
		want: `("test")`,
	}}
	for _, c := range cases {
		got := query.Print(c.q)
		if got != c.want {
			t.Errorf("got %s, want %s", got, c.want)
		}
	}
}

func TestEval_error(t *testing.T) {
	_, err := query.Eval(
		query.And(
			query.Or(
				query.Not("bar"),
				false),
			query.Not(true)),
		func(q query.Q) (*sqlf.Query, error) {
			return nil, errors.New("42")
		})
	if err == nil {
		t.Fatal("expected error")
	}
}
