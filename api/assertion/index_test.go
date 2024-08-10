package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	api "github.com/otakakot/vercel-go-passkey/api/assertion"
	"github.com/otakakot/vercel-go-passkey/pkg/testx"
)

func TestAssertionHandler(t *testing.T) {
	pwd, _ := os.Getwd()

	ddl := strings.Replace(pwd, "api/assertion", "schema", 1)

	testx.SetupPostgres(t, ddl)

	testx.SetupRedis(t)

	type want struct {
		status int
	}

	type args struct {
		rw  *httptest.ResponseRecorder
		req *http.Request
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "GET",
			args: args{
				rw:  httptest.NewRecorder(),
				req: httptest.NewRequest(http.MethodGet, "/assertion", nil),
			},
			want: want{
				status: http.StatusOK,
			},
		},
		{
			name: "DELETE",
			args: args{
				rw:  httptest.NewRecorder(),
				req: httptest.NewRequest(http.MethodDelete, "/assertion", nil),
			},
			want: want{
				status: http.StatusMethodNotAllowed,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api.Handler(tt.args.rw, tt.args.req)
			if tt.args.rw.Code != tt.want.status {
				t.Errorf("got: %v, want: %v. msg: %s", tt.args.rw.Code, tt.want.status, tt.args.rw.Body.String())
			}
		})
	}
}
