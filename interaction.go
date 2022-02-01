package slackmux

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

type BlockActionID string

type InteractionMux struct {
	vsm map[slack.InteractionType]map[string]viewSubmissionMuxEntry // view_submissionを処理するハンドラのマップ
	bam map[BlockActionID]blockActionMuxEntry
	slackMux
}

func NewInteractionMux(httpClient *http.Client) *InteractionMux {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &InteractionMux{slackMux: slackMux{httpClient: httpClient}}
}

type BlockActionHandlerFunc func(interactionCallback slack.InteractionCallback, blockAction *slack.BlockAction) (*slack.WebhookMessage, error)
type blockActionMuxEntry struct {
	h               BlockActionHandlerFunc
	interactionType slack.InteractionType
	actionID        BlockActionID
}

func (mux *InteractionMux) HandleBlockAction(actionID BlockActionID, handler BlockActionHandlerFunc) {
	if actionID == "" {
		panic("slack interaction mux: empty action id")
	}
	if handler == nil {
		panic("slack interaction mux: nil handler")
	}
	if _, exist := mux.bam[actionID]; exist {
		panic("slack interaction mux: multiple registration for " + actionID)
	}

	e := blockActionMuxEntry{h: handler, actionID: actionID}

	if mux.bam == nil {
		mux.bam = make(map[BlockActionID]blockActionMuxEntry)
	}

	mux.bam[actionID] = e
}

func (mux *InteractionMux) BlockActionsHandlerFunc(actionID BlockActionID) (BlockActionHandlerFunc, bool) {
	e, ok := mux.bam[actionID]
	if !ok {
		return nil, false
	}

	return e.h, true
}

type ViewSubmissionHandlerFunc func(interactionCallback slack.InteractionCallback) (*slack.ViewSubmissionResponse, error)

type viewSubmissionMuxEntry struct {
	h               ViewSubmissionHandlerFunc
	interactionType slack.InteractionType
	callbackID      string
}

func (mux *InteractionMux) HandleViewSubmission(interactionType slack.InteractionType, callbackID string, handler ViewSubmissionHandlerFunc) {
	if interactionType == "" {
		panic("slack interaction mux: empty interaction type")
	}
	if handler == nil {
		panic("slack interaction mux: nil handler")
	}
	if _, typeExist := mux.vsm[interactionType]; typeExist {
		if _, callbackIDExist := mux.vsm[interactionType][callbackID]; callbackIDExist {
			panic("slack interaction mux: multiple registration for " + interactionType)
		}
	}

	e := viewSubmissionMuxEntry{h: handler, interactionType: interactionType, callbackID: callbackID}

	if mux.vsm == nil {
		mux.vsm = make(map[slack.InteractionType]map[string]viewSubmissionMuxEntry)
	}

	if _, exist := mux.vsm[interactionType]; !exist {
		mux.vsm[interactionType] = make(map[string]viewSubmissionMuxEntry)
	}

	mux.vsm[interactionType][callbackID] = e
}

func (mux *InteractionMux) ViewSubmissionHandlerFunc(payload slack.InteractionCallback) (ViewSubmissionHandlerFunc, bool) {
	m, ok := mux.vsm[payload.Type]
	if !ok {
		return nil, false
	}

	e, ok := m[payload.View.CallbackID]
	if !ok {
		return nil, false
	}

	return e.h, true
}

func (mux *InteractionMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	payload := slack.InteractionCallback{}
	if err := payload.UnmarshalJSON([]byte(r.PostForm.Get("payload"))); err != nil {
		if mux.parseErrorHandlerFunc != nil {
			mux.parseErrorHandlerFunc(w, r, err)
		}
		return
	}

	switch payload.Type {
	case "block_actions":
		log.Printf("block_actions: %s", r.PostForm.Get("payload"))
		for _, blockAction := range payload.ActionCallback.BlockActions {
			h, exist := mux.BlockActionsHandlerFunc(BlockActionID(blockAction.ActionID))
			if exist {
				webhookMessage, err := h(payload, blockAction)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				// response_urlがあればそこにwebhook messageを送る
				if payload.ResponseURL != "" {
					b, err := json.Marshal(*webhookMessage)
					if err != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}

					req, err := http.NewRequest("POST", payload.ResponseURL, bytes.NewBuffer(b))
					req.Header.Add("Content-Type", "application/json")
					if err != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					if _, err := mux.httpClient.Do(req); err != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
				}
			}
		}
	case "view_submission":
		h, exist := mux.ViewSubmissionHandlerFunc(payload)
		if exist {
			resp, err := h(payload)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if resp == nil {
				return
			}
			b, _ := json.Marshal(resp)
			w.Header().Add("Content-Type", "application/json")
			w.Write(b)
			return
		}
	}
	// 該当するpayload.Typeが見つからなければエラー
	if mux.commandErrorHandlerFunc != nil {
		mux.commandErrorHandlerFunc(w, r, ErrCommandNotFound)
	}
}
