package slackmux

import "net/http"

type slackMux struct {
	parseErrorHandlerFunc        ErrorHandlerFunc
	verificationErrorHandlerFunc ErrorHandlerFunc
	commandErrorHandlerFunc      ErrorHandlerFunc
	httpClient                   *http.Client
}

func (mux *slackMux) HandleParseError(handler ErrorHandlerFunc) {
	mux.parseErrorHandlerFunc = handler
}

func (mux *slackMux) HandleVerificationError(handler ErrorHandlerFunc) {
	mux.verificationErrorHandlerFunc = handler
}

func (mux *slackMux) HandleCommandError(handler ErrorHandlerFunc) {
	mux.commandErrorHandlerFunc = handler
}
