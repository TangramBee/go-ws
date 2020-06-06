package http

import (
	"go-ws/utils/errs"
	"go-ws/utils/logger"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/buger/jsonparser"
)

func Post(url string, data url.Values) (err error) {
	var rsp *http.Response
	rsp, err = http.PostForm(url, data)
	if err != nil {
		logger.Logger.Warn("push msg api request failed", zap.String("url", url), zap.Any("data", data), zap.Error(err))
		return
	}

	defer rsp.Body.Close()

	var respBody []byte
	respBody, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		logger.Logger.Warn("push msg api request failed", zap.String("url", url), zap.Any("data", data), zap.Error(err))
		return
	}

	var code int64
	code, err = jsonparser.GetInt(respBody, "code")
	if err != nil {
		logger.Logger.Warn("push msg api request failed", zap.String("url", url), zap.Any("data", data), zap.Any("response", respBody), zap.Error(err))
		return
	}

	if code != 0 {
		logger.Logger.Warn("push msg api request failed", zap.String("url", url), zap.Any("data", data), zap.Any("response", respBody), zap.Error(err))
		return errs.ErrRequestUrlFailed
	}

	return nil
}



