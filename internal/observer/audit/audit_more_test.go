package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewNotifier принимает URL и отправляет событие POST без паники.
func TestNotifier_HTTPSink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	n, err := NewNotifier("", srv.URL)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, n.Close()) })

	n.Notify(NewEvent(ActionLogin, "user-1", "", ""))
}

// Пустые пути — без файла: Close не падает.
func TestNotifier_Empty_Close(t *testing.T) {
	n, err := NewNotifier("", "")
	require.NoError(t, err)
	assert.NoError(t, n.Close())
}
