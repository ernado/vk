package vk

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHttpClient(t *testing.T) {
	client := getDefaultHTTPClient()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer server.Close()

	Convey("Test", t, func() {
		req, err := http.NewRequest("GET", server.URL, nil)
		So(err, ShouldBeNil)
		res, err := client.Do(req)
		So(err, ShouldBeNil)
		So(res.StatusCode, ShouldEqual, http.StatusNotFound)
	})
}

func BenchmarkDummyEncoder(b *testing.B) {
	type Data struct {
		Error    `json:"error"`
		Response []struct {
			ID int64 `json:"id"`
		} `json:"response"`
	}
	value := Data{}
	for i := 0; i < b.N; i++ {
		sData := `{
			"response": [
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]
		}`
		body := ioutil.NopCloser(bytes.NewBufferString(sData))
		json.NewDecoder(body).Decode(&value)
	}
}

func BenchmarkVKEncoder(b *testing.B) {
	type Data struct {
		Error    `json:"error"`
		Response []struct {
			ID int64 `json:"id"`
		} `json:"response"`
	}
	value := Data{}
	for i := 0; i < b.N; i++ {
		sData := `{
			"response": [
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]
		}`
		body := ioutil.NopCloser(bytes.NewBufferString(sData))
		Process(body).To(&value)
	}
}

func TestMust(t *testing.T) {
	Convey("Must Panic", t, func() {
		err := ErrUnknown
		So(func() {
			must(err)
		}, ShouldPanicWith, ErrUnknown)
	})
}

func TestNewRequest(t *testing.T) {
	Convey("New request", t, func() {
		values := url.Values{}
		values.Add("foo", "bar")
		r := Request{Token: "token", Method: "users.get", Values: values}
		req := r.HTTP()
		So(req.URL.Host, ShouldEqual, defaultHost)
		So(req.URL.String(), ShouldEqual, "https://api.vk.com/method/users.get?access_token=token&foo=bar&https=1&v=5.35")
	})
}

type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e errorReader) Close() error {
	return nil
}

func TestResponseProcessor(t *testing.T) {
	Convey("Decoder", t, func() {
		sData := `{
			"response": [
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]
		}`
		body := ioutil.NopCloser(bytes.NewBufferString(sData))
		type Data struct {
			Error    `json:"error"`
			Response []struct {
				ID int64 `json:"id"`
			} `json:"response"`
		}
		So(Process(body).To(&Data{}), ShouldBeNil)
		Convey("Read error", func() {
			So(Process(errorReader{}).To(&Data{}), ShouldNotBeNil)
		})
		Convey("Error", func() {
			sData := `{"error":{"error_code":10,"error_msg":"Internal server error: could not get application",
			"request_params":[{"key":"oauth","value":"1"},{"key":"method","value":"users.get"},
			{"key":"user_id","value":"1"},{"key":"v","value":"5.35"}]}}`
			body := bytes.NewBufferString(sData)
			value := Data{}
			err := Process(body).To(&value)
			So(err, ShouldNotBeNil)
			So(IsServerError(err), ShouldBeTrue)
			serverError := GetServerError(err)
			So(serverError.Code, ShouldEqual, ErrInternalServerError)
		})
	})
}

func TestRawResponseProcessor(t *testing.T) {
	Convey("Encode", t, func(){
		data, err := Raw{1}.MarshalJSON()
		So(err, ShouldBeNil)
		So(data[0], ShouldEqual, 1)
	})
	Convey("Decoder", t, func() {
		sData := `{
			"response": [
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]
		}`
		body := ioutil.NopCloser(bytes.NewBufferString(sData))
		type Data []struct {
			ID int64 `json:"id"`
		}
		var value Data
		So(Process(body).Raw(&value), ShouldBeNil)
		Convey("Read error", func() {
			So(Process(errorReader{}).Raw(&value), ShouldNotBeNil)
		})
		Convey("Error", func() {
			sData := `{"error":{"error_code":10,"error_msg":"Internal server error: could not get application",
			"request_params":[{"key":"oauth","value":"1"},{"key":"method","value":"users.get"},
			{"key":"user_id","value":"1"},{"key":"v","value":"5.35"}]}}`
			body := bytes.NewBufferString(sData)
			value := Data{}
			err := Process(body).Raw(&value)
			So(err, ShouldNotBeNil)
			So(IsServerError(err), ShouldBeTrue)
			serverError := GetServerError(err)
			So(serverError.Code, ShouldEqual, ErrInternalServerError)
		})
	})
}

func TestRequestSerialization(t *testing.T) {
	Convey("New request", t, func() {
		values := url.Values{}
		values.Add("foo", "bar")
		r := Request{Token: "token", Method: "users.get", Values: values}
		req := r.HTTP()
		So(req.URL.Host, ShouldEqual, defaultHost)
		So(req.URL.String(), ShouldEqual, "https://api.vk.com/method/users.get?access_token=token&foo=bar&https=1&v=5.35")

		Convey("Marshal ok", func() {
			data, err := json.Marshal(r)
			So(err, ShouldBeNil)
			So(bytes.NewBuffer(data).String(), ShouldEqual, `{"method":"users.get","token":"token","values":{"foo":["bar"]}}`)

			Convey("Consistent after unmarshal", func() {
				newRequest := Request{}
				So(json.Unmarshal(data, &newRequest), ShouldBeNil)
			})
		})
	})
}

type simpleHTTPClientMock struct {
	response *http.Response
	err      error
}

type apiClientMock struct {
	callback func(Request, Response) error
}

func (client apiClientMock) Do(request Request, response Response) error {
	return client.callback(request, response)
}

func (m simpleHTTPClientMock) Do(request *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestDo(t *testing.T) {
	client := New()

	Convey("Do request", t, func() {
		sData := `{
			"response": [
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]
		}`
		body := ioutil.NopCloser(bytes.NewBufferString(sData))
		httpResponse := &http.Response{Body: body, StatusCode: http.StatusOK}
		client.SetHTTPClient(simpleHTTPClientMock{response: httpResponse})
		type Data struct {
			Error    `json:"error"`
			Response []struct {
				ID   int64  `json:"id"`
				Name string `json:"first_name"`
			} `json:"response"`
		}

		request := Request{Method: "users.get"}
		response := &Data{}

		So(client.Do(request, response), ShouldBeNil)
		So(response.Response[0].ID, ShouldEqual, 1)
		So(response.Response[0].Name, ShouldEqual, "Павел")
		So(response.Request.Method, ShouldEqual, "users.get")

		Convey("Bad status", func() {
			client := New()
			httpResponse := &http.Response{Body: body, StatusCode: http.StatusBadRequest}
			client.SetHTTPClient(simpleHTTPClientMock{response: httpResponse})
			request := Request{Method: "users.get"}
			response := &Data{}

			So(client.Do(request, response), ShouldEqual, ErrBadResponseCode)
		})
		Convey("Http error", func() {
			client := New()
			httpResponse := &http.Response{Body: body, StatusCode: http.StatusBadRequest}
			client.SetHTTPClient(simpleHTTPClientMock{response: httpResponse, err: ErrBadResponseCode})
			request := Request{Method: "users.get"}
			response := &Data{}

			So(client.Do(request, response), ShouldEqual, ErrBadResponseCode)
		})
	})
}


func TestDoRawResponse(t *testing.T) {
	client := New()

	Convey("Do request", t, func() {
		sData := `{
			"response": [
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]
		}`
		body := ioutil.NopCloser(bytes.NewBufferString(sData))
		httpResponse := &http.Response{Body: body, StatusCode: http.StatusOK}
		client.SetHTTPClient(simpleHTTPClientMock{response: httpResponse})

		request := Request{Method: "users.get"}
		response := &RawResponse{}

		So(client.Do(request, response), ShouldBeNil)
		expectedRaw := `[
				{
					"id": 1,
					"first_name": "Павел",
					"last_name": "Дуров"
				}
			]`
		So(expectedRaw, ShouldEqual, response.Response.String())
	})
}
