package pipeline

import (
	"errors"
	"testing"

	"github.com/lumen/relay/internal/provider"
)

func TestNarrowRetry(t *testing.T) {
	ok, _ := ShouldRetry(provider.Result{Transient: true, Code: "421"}, errors.New("tmp"), 1)
	if !ok {
		t.Fatal("4xx retry")
	}
	ok, _ = ShouldRetry(provider.Result{AuthFailed: true, Code: "535"}, errors.New("auth"), 1)
	if ok {
		t.Fatal("auth must not retry")
	}
	ok, _ = ShouldRetry(provider.Result{Code: "550", Message: "user unknown"}, errors.New("550"), 1)
	if ok {
		t.Fatal("hard bounce no retry")
	}
	ok, _ = ShouldRetry(provider.Result{Transient: true}, errors.New("x"), 3)
	if ok {
		t.Fatal("max attempts")
	}
}
