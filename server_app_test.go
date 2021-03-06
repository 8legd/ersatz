package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var defaultJSON = `{ "response_code": 200, "headers": { "header-1": "some value" }, "body": { "a":1, "b":2, "c":3 }}`

type definitionFile struct {
	verb      string
	endpoints []string
	variants  []string
	json      []string
}

func Test_ItReturnsErrorIfPathDoesntExist(t *testing.T) {

	if err := NewServerApp("9999", "/this/doesnt/exist").Setup(); err == nil {
		t.Error("Able to read from a non-existent directory")
	}
}

func Test_ItSetsupAHandlerForAllTheFilesOnThePath(t *testing.T) {
	dirName, cleanupFn := setupRootDir(t)
	defer cleanupFn()

	endpoints := []definitionFile{
		{HTTP_POST, []string{"endpoint1"}, []string{"default.json"}, []string{defaultJSON}},
		{HTTP_GET, []string{"endpoint1"}, []string{"default.json"}, []string{defaultJSON}},
		{HTTP_GET, []string{"endpoint1", "subendpoint1"}, []string{"default.json"}, []string{defaultJSON}},
	}

	setupSubFolders(t, dirName, endpoints)
	setupDefinitionFiles(t, dirName, endpoints)

	startApp := NewServerApp("9998", dirName)

	err := startApp.Setup()
	assert.Nil(t, err)

	exit := make(chan interface{})

	go startApp.Run(exit)

	for _, df := range endpoints {

		path := strings.Join(df.endpoints, "/")

		req, err := http.NewRequest(
			df.verb,
			fmt.Sprintf("http://localhost:%s/%s", "9998", path),
			bytes.NewBuffer([]byte("")),
		)

		assert.Nil(t, err)

		client := &http.Client{}
		res, err := client.Do(req)
		assert.Nil(t, err)

		if g, e := res.StatusCode, 200; g != e {
			t.Errorf("Expected status code %d, got %d", e, g)
		}
	}

	exit <- true
}

func Test_ItReturnsTheRightJSONAndHeadersForFiles(t *testing.T) {

	dirName, cleanupFn := setupRootDir(t)
	defer cleanupFn()

	// Expected results
	expectedHeaders := map[string]string{
		"Header-One": "Value 1",
		"Header-Two": "Value 2",
	}

	expectedBody := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	// Define specific JSON for the endpoint
	ep := NewEndpoint()

	for k, v := range expectedHeaders {
		ep.Headers[k] = v
	}

	ep.ResponseCode = 201

	ep.Body = expectedBody

	raw_json, err := json.Marshal(ep)
	assert.Nil(t, err)

	endpoint := definitionFile{
		HTTP_POST,
		[]string{"endpoint"},
		[]string{"default.json"},
		[]string{string(raw_json)},
	}

	// Create the files for the endpoint
	setupSubFolders(t, dirName, []definitionFile{endpoint})
	setupDefinitionFiles(t, dirName, []definitionFile{endpoint})

	// Start and set up the app
	startApp := NewServerApp("9998", dirName)

	serr := startApp.Setup()
	assert.Nil(t, serr)

	// Run the app
	exit := make(chan interface{})
	go startApp.Run(exit)

	// Make a request to the endpoint
	path := strings.Join(endpoint.endpoints, "/")

	req, err := http.NewRequest(
		endpoint.verb,
		fmt.Sprintf("http://localhost:%s/%s", "9998", path),
		bytes.NewBuffer([]byte("")),
	)

	assert.Nil(t, err)

	client := &http.Client{}
	res, err := client.Do(req)
	assert.Nil(t, err)

	// Make sure thet everything comes out as expected
	if g, e := res.StatusCode, ep.ResponseCode; g != e {
		t.Errorf("Expected status code %d, got %d", e, g)
	}

	for k, v := range expectedHeaders {
		assert.Equal(t, res.Header.Get(k), v)
	}

	// Check the body
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	bodyResult := make(map[string]int)

	assert.Nil(t, json.Unmarshal(body, &bodyResult))

	assert.True(t, reflect.DeepEqual(expectedBody, bodyResult))

	// Close the server
	exit <- true
}

func Test_TheControlEndpointResponds400ToInvalidRequests(t *testing.T) {

	// Start and set up the app
	startApp := NewServerApp("9998", ".")

	serr := startApp.Setup()
	assert.Nil(t, serr)

	// Run the app
	exit := make(chan interface{})
	go startApp.Run(exit)

	req, err := http.NewRequest(
		HTTP_POST,
		"http://localhost:9998/__ersatz",
		bytes.NewBuffer([]byte("")),
	)

	assert.Nil(t, err)

	client := &http.Client{}
	res, err := client.Do(req)
	assert.Nil(t, err)

	// Make sure thet everything comes out as expected
	if g, e := res.StatusCode, 400; g != e {
		t.Errorf("Expected status code %d, got %d", e, g)
	}

	// Close the server
	exit <- true
}

func Test_TheControlEndpointResponds200ToAValidVaryRequest(t *testing.T) {

	// Create a command
	cmd := NewServerCommand()

	cmd.Command = SERVER_COMMAND_VARY
	cmd.VariableEndpointIndex = VariableEndpointIndex{
		EndpointIndex{
			URL:    "some/endpoint",
			Method: HTTP_GET,
		},
		"some_variation",
	}

	raw_json, err := json.Marshal(cmd)
	assert.Nil(t, err)

	// Start and set up the app
	startApp := NewServerApp("9998", ".")

	serr := startApp.Setup()
	assert.Nil(t, serr)

	// Run the app
	exit := make(chan interface{})
	go startApp.Run(exit)

	req, err := http.NewRequest(
		HTTP_POST,
		"http://localhost:9998/__ersatz",
		bytes.NewBuffer(raw_json),
	)

	assert.Nil(t, err)

	client := &http.Client{}
	res, err := client.Do(req)
	assert.Nil(t, err)

	// Make sure thet everything comes out as expected
	if g, e := res.StatusCode, 200; g != e {
		t.Errorf("Expected status code %d, got %d", e, g)
	}

	// Close the server
	exit <- true
}

func Test_AnEndpointCanBeVaried(t *testing.T) {

	dirName, cleanupFn := setupRootDir(t)
	defer cleanupFn()

	// Expected results
	expectedHeaders := map[string]string{
		"Header-One": "Value 1",
		"Header-Two": "Value 2",
	}

	expectedBody := map[string]int{
		"d": 4,
		"b": 5,
		"e": 6,
	}

	// Define specific JSON for the endpoint
	ep := NewEndpoint()

	for k, v := range expectedHeaders {
		ep.Headers[k] = v
	}

	ep.ResponseCode = 201

	ep.Body = expectedBody

	raw_json, err := json.Marshal(ep)
	assert.Nil(t, err)

	endpoint := definitionFile{
		HTTP_POST,
		[]string{"endpoint"},
		[]string{"default.json", "some_variation.json"},
		[]string{defaultJSON, string(raw_json)},
	}

	// Create the files for the endpoint
	setupSubFolders(t, dirName, []definitionFile{endpoint})
	setupDefinitionFiles(t, dirName, []definitionFile{endpoint})

	// Create a command
	cmd := NewServerCommand()

	cmd.Command = SERVER_COMMAND_VARY
	cmd.VariableEndpointIndex = VariableEndpointIndex{
		EndpointIndex{
			URL:    "endpoint",
			Method: HTTP_POST,
		},
		"some_variation",
	}

	raw_json, jerr := json.Marshal(cmd)
	assert.Nil(t, jerr)

	// Start and set up the app
	startApp := NewServerApp("9996", dirName)

	serr := startApp.Setup()
	assert.Nil(t, serr)

	// Run the app
	exit := make(chan interface{})
	go startApp.Run(exit)

	// Make the control request
	req, err := http.NewRequest(
		HTTP_POST,
		"http://localhost:9996/__ersatz",
		bytes.NewBuffer(raw_json),
	)

	assert.Nil(t, err)

	client := &http.Client{}
	_, cerr := client.Do(req)
	assert.Nil(t, cerr)

	// Make the request, which should now be varied
	path := strings.Join(endpoint.endpoints, "/")

	req2, err := http.NewRequest(
		endpoint.verb,
		fmt.Sprintf("http://localhost:%s/%s", "9996", path),
		bytes.NewBuffer([]byte("")),
	)

	assert.Nil(t, err)

	client2 := &http.Client{}
	res, err := client2.Do(req2)
	assert.Nil(t, err)

	// Make sure thet everything comes out as expected
	if g, e := res.StatusCode, ep.ResponseCode; g != e {
		t.Errorf("Expected status code %d, got %d", e, g)
	}

	for k, v := range expectedHeaders {
		assert.Equal(t, res.Header.Get(k), v)
	}

	// Check the body
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	bodyResult := make(map[string]int)

	assert.Nil(t, json.Unmarshal(body, &bodyResult))

	assert.True(t, reflect.DeepEqual(expectedBody, bodyResult))

	// Close the server
	exit <- true
}

func setupDefinitionFiles(t *testing.T, rootDir string, dfs []definitionFile) {
	for _, df := range dfs {

		for i, variant := range df.variants {

			path := filepath.Join(rootDir, filepath.Join(df.endpoints...), df.verb) + "/" + variant

			// Create the file itself
			f, err := os.Create(path)
			assert.Nil(t, err)

			// Write some stock data into it
			n, err := f.Write([]byte(df.json[i]))
			assert.Nil(t, err)
			assert.Equal(t, n, len(df.json[i]))
		}
	}
}

func setupSubFolders(t *testing.T, rootDir string, dfs []definitionFile) {
	for _, df := range dfs {

		path := filepath.Join(rootDir, filepath.Join(df.endpoints...), df.verb)
		os.MkdirAll(path, 0755)
	}
}

func setupRootDir(t *testing.T) (string, func()) {
	dirName, err := ioutil.TempDir("", "ersatz")
	assert.Nil(t, err)

	return dirName, func() {
		if err := os.RemoveAll(dirName); err != nil {
			panic(err.Error())
		}
	}
}
