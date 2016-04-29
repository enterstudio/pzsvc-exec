// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pzsvc

// TODO: Will probably want to rename/rearrange/refactor the pzsvc-exec package so as to better conform
// to go coding standards/naming conventions at some point.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	//"strings"
	"time"
)

type JobResp struct {
	Type  string
	JobId string
}

type StatusJsonResult struct {
	Id               string
	Name             string
	Type             string
	DataId           string
	Message          string
	Details          string
	ResourceMetadata map[string]string
}

type StatusJsonResp struct {
	Type    string
	JobId   string
	Result  StatusJsonResult
	Results []StatusJsonResult
	Status  string
}

func submitMultipart(bodyStr, runId, address, upload string) (*http.Response, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("body", bodyStr)
	if err != nil {
		return nil, err
	}

	if upload != "" {
		file, err := os.Open(fmt.Sprintf(`./%s/%s`, runId, upload))
		if err != nil {
			return nil, err
		}

		defer file.Close()

		part, err := writer.CreateFormFile("file", upload)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	fileReq, err := http.NewRequest("POST", address, body)
	if err != nil {
		return nil, err
	}

	fileReq.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func submitSinglePart(method, bodyStr, address string) (*http.Response, error) {

	fileReq, err := http.NewRequest(method, address, bytes.NewBuffer([]byte(bodyStr)))
	if err != nil {
		return nil, err
	}

    fileReq.Header.Add("Content-Type", "application/json")
    fileReq.Header.Add("size", "30")
    fileReq.Header.Add("from", "0")
    fileReq.Header.Add("key", "stamp")
    fileReq.Header.Add("order", "true")

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func Download(dataId, runId, pzAddr string) (string, error) {

	resp, err := http.Get(pzAddr + "/file/" + dataId)	
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}	
	
	contDisp := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contDisp)
	filename := params["filename"]
	if filename == "" {
		filename = "dummy.txt"
	}
	filepath := fmt.Sprintf(`./%s/%s`, runId, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

func getDataId(jobId, pzAddr string) (string, error) {

	type JobResult struct {
		Type			string
		DataId			string
	}
	
	type JobProg struct {
		PercentComplete	int
	}
	
	type JobResp struct {
		Type			string
		JobId			string
		Result			JobResult
		Status			string
		JobType			string
		Progress		JobProg
		Message			string
		Origin			string
	}

	time.Sleep(1000 * time.Millisecond)


	for i := 0; i < 100; i++ {
		
		resp, err := http.Get(pzAddr + "/job/" + jobId)	
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return "", err
		}

		respBuf := &bytes.Buffer{}

		_, err = respBuf.ReadFrom(resp.Body)
		if err != nil {
			return "", err
		}

		fmt.Println(respBuf.String())

		var respObj JobResp
		err = json.Unmarshal(respBuf.Bytes(), &respObj)
		if err != nil {
			return "", err
		}

		if respObj.Status == "Submitted" || respObj.Status == "Running" || respObj.Status == "Pending" || respObj.Message == "Job Not Found" {
			time.Sleep(200 * time.Millisecond)
		} else if respObj.Status == "Success" {
			return respObj.Result.DataId, nil
		} else if respObj.Status == "Error" || respObj.Status == "Fail" {
			return "", errors.New(respObj.Status + ": " + respObj.Message)
		} else {
			return "", errors.New("Unknown status: " + respObj.Status)
		}
	}

	return "", errors.New("Never completed.")
}

func ingestMultipart(bodyStr, runId, pzAddr, filename string) (string, error) {

	resp, err := submitMultipart(bodyStr, runId, (pzAddr + "/job"), filename)
	if err != nil {
		return "", err
	}

	respBuf := &bytes.Buffer{}

	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println(respBuf.String())

	var respObj JobResp
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Println("error:", err)
	}

	return getDataId(respObj.JobId, pzAddr)
}

func IngestTiff(filename, runId, pzAddr, cmdName string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "raster" }, "metadata": { "name": "%s", "description": "raster uploaded by pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, filename, cmdName)

	return ingestMultipart(jsonStr, runId, pzAddr, filename)
}

func IngestGeoJson(filename, runId, pzAddr, cmdName string) (string, error) {

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" : { "dataType": { "type": "geojson" }, "metadata": { "name": "%s", "description": "GeoJson uploaded by pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, filename, cmdName)

	return ingestMultipart(jsonStr, runId, pzAddr, filename)
}

func IngestTxt(filename, runId, pzAddr, cmdName string) (string, error) {
	textblock, err := ioutil.ReadFile(fmt.Sprintf(`./%s/%s`, runId, filename))
	if err != nil {
		return "", err
	}

	jsonStr := fmt.Sprintf(`{ "userName": "my-api-key-38n987", "jobType": { "type": "ingest", "host": "true", "data" :{ "dataType": { "type": "text", "mimeType": "application/text", "content": "%s" }, "metadata": { "name": "%s", "description": "text output from pzsvc-exec for %s.", "classType": { "classification": "unclassified" } } } } }`, strconv.QuoteToASCII(string(textblock)), filename, cmdName)

	return ingestMultipart(jsonStr, runId, pzAddr, "")
}

func RegisterSvc(svcName, svcDesc, svcUrl, pzAddr string) error {

	// TODO: add customizable metadata once the inputs are stabilized

	jsonStr := fmt.Sprintf(`{ "inputs": [], "outputs": [], "url": "%s/execute", "resourceMetadata": { "name": "%s", "description": "%s", "method": "POST" } }`, svcUrl, svcName, svcDesc)
fmt.Println(jsonStr)
	_, err := submitSinglePart("POST", jsonStr, pzAddr + "/service")
	if err != nil {
		return err
	}

	return nil
}

func UpdateSvc(svcName, svcId, svcDesc, svcUrl, pzAddr string) error {
	jsonStr := fmt.Sprintf(`{ "serviceId": "%s", "url": "%s/execute", "resourceMetadata": { "name": "%s", "description": "%s", "method": "POST" } }`, svcId, svcUrl, svcName, svcDesc)
	
	_, err := submitSinglePart("PUT", jsonStr, pzAddr + "/service/" + svcId)
	if err != nil {
		return err
	}

	return nil
}

// Searches Pz for a service matching the input information.  if it finds one,
// returns the service ID.  If it does not, returns an empty string.
// currently only semi-functional.  Read through and vet carefully before declaring functional.

func FindMySvc(svcName, pzAddr string) (string, error) {
	
	type SvcData struct {
		ServiceId			string
		Url					string
		ResourceMetadata	map[string]string
	}
	
	type SvcWrapper struct {
		Type		string
		Data		[]SvcData
		Pagination	map[string]int
		
	}

fmt.Println(pzAddr + "/service?per_page=1000&keyword=" + url.QueryEscape(svcName))

	resp, err := http.Get(pzAddr + "/service?per_page=1000&keyword=" + url.QueryEscape(svcName))	
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respObj SvcWrapper
	err = json.Unmarshal(respBuf, &respObj)
	if err != nil {
		return "", err
	}

	for _, checkServ := range respObj.Data {
		if checkServ.ResourceMetadata["name"] == svcName {
			return checkServ.ServiceId, nil
		}
	}

	return "", nil
}

func ManageRegistration(svcName, svcDesc, svcUrl, pzAddr string) error {
fmt.Println("Finding")
	svcId, err := FindMySvc(svcName, pzAddr)
	if err != nil {
		return err
	}

	if (svcId == "") {
fmt.Println("Registering")
		err = RegisterSvc(svcName, svcDesc, svcUrl, pzAddr)
	} else {
fmt.Println("Updating")
		err = UpdateSvc(svcName, svcId, svcDesc, svcUrl, pzAddr)
	}
	if err != nil {
		return err
	}

	return nil
}