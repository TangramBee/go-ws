package errs

import (
	"fmt"

)

// StandardError struct
type StandardError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (err StandardError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", err.Code, err.Msg)
}

// WithMsg is method to set error msg
func (err StandardError) WithMsg(msg string) StandardError {
	err.Msg = err.Msg + ": " + msg
	return err
}

// New StarandError
func New(code int, msg string) StandardError {
	return StandardError{code, msg}
}

var (
	Success    = StandardError{0, "success"}
	ErrUnknown = StandardError{10001, "unknown error"}
	ErrParam   = StandardError{10002, "params is invalid"}

	//login err
	ErrLoginError   = StandardError{10003, "login error"}   //登录失败
	ErrLoginExpired = StandardError{10004, "login expired"} //登录过期
	ErrUnLogin      = StandardError{10005, "unlogin"}       //未登录
	ErrRequestUrlFailed      = StandardError{10006, "request url fauled"}       //未登录

	ErrPushMsgToQueueFailed   = StandardError{20001, "push msg to queue failed"}
	ErrWebSocketHaveOtherConnection = StandardError{20002, "websocket already connected in elsewhere"}
	ErrWebSocketConnectionIDIsNil      = StandardError{20003, "websocket connection id is nil"}
	ErrWebSocketMessageIsNone       = StandardError{20004, "websocket send message is null"}

)
