package ecloudcoding

import (
	"net/http"
	"reflect"
	"testing"
)

func TestModels_ReturnsStaticCatalogList(t *testing.T) {
	c := NewClient(http.DefaultClient, "", "test-key")
	models, err := c.Models()
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	want := []string{"cm-code-latest", "minimax-m2.5"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("unexpected models: got=%v want=%v", models, want)
	}
}
