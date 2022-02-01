package slackmux_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kecbigmt/slackmux"
	"github.com/slack-go/slack"
)

var (
	normalPayload = `{
    "type": "view_submission",
    "team": { "id": "TEAM_ID", "domain": "DOMAIN" },
    "user": {
      "id": "ID",
      "username": "USERNAME",
      "name": "NAME",
      "team_id": "TEAM_ID"
    },
    "api_app_id": "API_APP_ID",
    "token": "TOKEN",
    "trigger_id": "TRIGGER_ID",
    "view": {
      "id": "VIEW_ID",
      "team_id": "TEAM_ID",
      "type": "modal",
      "blocks": [],
      "private_metadata": "",
      "callback_id": "normal-callback",
      "state": {},
      "hash": "HASH",
      "title": {
        "type": "plain_text",
        "text": "TEXT",
        "emoji": true
      },
      "clear_on_close": false,
      "notify_on_close": false,
      "close": null,
      "submit": { "type": "plain_text", "text": "TEXT", "emoji": true },
      "previous_view_id": null,
      "root_view_id": "ROOT_VIEW_ID",
      "app_id": "APP_ID",
      "external_id": "",
      "app_installed_team_id": "APP_INSTALLED_TEAM_ID",
      "bot_id": "BOT_ID"
    },
    "response_urls": [],
    "is_enterprise_install": false,
    "enterprise": null
  }`
	unknownCallbackPayload = `{
    "type": "view_submission",
    "team": { "id": "TEAM_ID", "domain": "DOMAIN" },
    "user": {
      "id": "ID",
      "username": "USERNAME",
      "name": "NAME",
      "team_id": "TEAM_ID"
    },
    "api_app_id": "API_APP_ID",
    "token": "TOKEN",
    "trigger_id": "TRIGGER_ID",
    "view": {
      "id": "VIEW_ID",
      "team_id": "TEAM_ID",
      "type": "modal",
      "blocks": [],
      "private_metadata": "",
      "callback_id": "unknown-callback",
      "state": {},
      "hash": "HASH",
      "title": {
        "type": "plain_text",
        "text": "TEXT",
        "emoji": true
      },
      "clear_on_close": false,
      "notify_on_close": false,
      "close": null,
      "submit": { "type": "plain_text", "text": "TEXT", "emoji": true },
      "previous_view_id": null,
      "root_view_id": "ROOT_VIEW_ID",
      "app_id": "APP_ID",
      "external_id": "",
      "app_installed_team_id": "APP_INSTALLED_TEAM_ID",
      "bot_id": "BOT_ID"
    },
    "response_urls": [],
    "is_enterprise_install": false,
    "enterprise": null
  }`
)

func createPostBody(payload string) *bytes.Buffer {
	values := url.Values{}
	values.Set("payload", payload)
	return bytes.NewBufferString(values.Encode())
}

func TestViewSubmission(t *testing.T) {
	mux := slackmux.NewInteractionMux(nil)
	mux.HandleParseError(func(w http.ResponseWriter, r *http.Request, err error) {
		w.Write([]byte("parse error"))
	})
	mux.HandleCommandError(func(w http.ResponseWriter, r *http.Request, err error) {
		w.Write([]byte("command error"))
	})
	mux.HandleViewSubmission("view_submission", "normal-callback", func(interactionCallback slack.InteractionCallback) (*slack.ViewSubmissionResponse, error) {
		return slack.NewClearViewSubmissionResponse(), nil
	})

	type args struct {
		requestBody *bytes.Buffer
	}
	test := []struct {
		name string
		want string
		args args
	}{
		{
			name: "parse error",
			want: "parse error",
			args: args{requestBody: bytes.NewBufferString("")},
		},
		{
			name: "command error",
			want: "command error",
			args: args{requestBody: createPostBody(unknownCallbackPayload)},
		},
		{
			name: "normal",
			want: `{"response_action":"clear"}`,
			args: args{requestBody: createPostBody(normalPayload)},
		},
	}

	for _, tt := range test {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "http://dummy.com/interactive-endpoint", tt.args.requestBody)
		r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		t.Run(tt.name, func(t *testing.T) {
			mux.ServeHTTP(w, r)
			if got := w.Body.String(); got != tt.want {
				t.Fatalf("%s: got '%s', want '%s'", tt.name, got, tt.want)
			}
		})
	}
}
