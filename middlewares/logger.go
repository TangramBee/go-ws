package middlewares

import (
	"bytes"
	"encoding/json"
	"go-ws/utils/logger"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func RequestLogger() gin.HandlerFunc {

	return func(c *gin.Context) {
		traceID := uuid.New().String()

		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
		}
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		var response interface{}
		if respMap, err := formatResponse(blw.body.Bytes()); err == nil {
			response = respMap
		} else {
			response = blw.body.Bytes()
		}

		var bodyStr string
		if len(bodyBytes) > 0 {
			bodyStr, _ = url.QueryUnescape(string(bodyBytes))
		}

		var cookies = formatCookies(c.Request.Cookies())
		if agentID, ok := c.Get("agent_id"); ok {
			cookies["agent_id"] = strconv.Itoa(int(agentID.(int32)))
			cookies["agent_name"] = c.GetString("agent_name")
		}
		logger.Logger.Info(
			"access_log",
			zap.String("id", traceID),
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
			zap.String("body", bodyStr),
			zap.Any("cookies", cookies),
			zap.Any("response", response),
		)

		if len(c.Errors) > 0 {
			logger.Logger.Error(c.Errors.ByType(gin.ErrorTypeAny).String())
		}
	}
}

func formatCookies(cookies []*http.Cookie) (cookiesMap map[string]string) {
	cookiesMap = make(map[string]string)
	for _, cookie := range cookies {
		cookiesMap[cookie.Name] = cookie.Value
	}
	return
}

func formatResponse(data []byte) (map[string]interface{}, error) {
	var responseMap = make(map[string]interface{})
	err := json.Unmarshal(data, &responseMap)
	if err != nil {
		return responseMap, err
	}
	return responseMap, nil
}
