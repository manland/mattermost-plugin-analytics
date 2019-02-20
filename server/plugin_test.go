package main

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeHTTP(t *testing.T) {
	assert := assert.New(t)
	plugin := Plugin{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/pie.svg?chan1=1&chan2=1", nil)

	plugin.ServeHTTP(nil, w, r)

	result := w.Result()
	assert.NotNil(result)
	bodyBytes, err := ioutil.ReadAll(result.Body)
	assert.Nil(err)
	bodyString := string(bodyBytes)

	assert.Equal("<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" width=\"300\" height=\"300\">\\n<path  d=\"M 0 0\nL 300 0\nL 300 300\nL 0 300\nL 0 0\" style=\"stroke-width:0;stroke:rgba(255,255,255,1.0);fill:rgba(255,255,255,1.0)\"/><path  d=\"M 5 5\nL 295 5\nL 295 295\nL 5 295\nL 5 5\" style=\"stroke-width:0;stroke:rgba(255,255,255,1.0);fill:rgba(255,255,255,1.0)\"/><path  d=\"M 150 150\nL 295 150\nA 145 145 180.00 0 1 5 150\nL 150 150\nZ\" style=\"stroke-width:5;stroke:rgba(255,255,255,1.0);fill:rgba(106,195,203,1.0)\"/><path  d=\"M 150 150\nL 5 150\nA 145 145 180.00 0 1 295 150\nL 150 150\nZ\" style=\"stroke-width:5;stroke:rgba(255,255,255,1.0);fill:rgba(42,190,137,1.0)\"/><text x=\"129\" y=\"253\" style=\"stroke-width:0;stroke:none;fill:rgba(51,51,51,1.0);font-size:15.3px;font-family:'Roboto Medium',sans-serif\">chan1</text><text x=\"129\" y=\"61\" style=\"stroke-width:0;stroke:none;fill:rgba(51,51,51,1.0);font-size:15.3px;font-family:'Roboto Medium',sans-serif\">chan2</text></svg>", bodyString)
}
