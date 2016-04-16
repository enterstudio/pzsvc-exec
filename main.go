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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type ConfigType struct {
	CliCmd string
	PzAddr string
	SvcName string
	SvcType string
	Port int
	Description string
	Attributes interface {}
}

type OutStruct struct {
    InFiles map[string] string
    OutFiles map[string] string
	ProgReturn string
}

func main() {

	// first argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configBuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil { fmt.Println("error:", err) }

	var configObj ConfigType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil { fmt.Println("error:", err) }

	//- check that config file data is complete.  Checks other dependency requirements (if any)
	//- register on Pz
	
	if configObj.SvcName != "" && configObj.PzAddr != "" {
		pzsvc.ManageRegistration(configObj.SvcName, configObj.SvcType, configObj.PzAddr)
	}
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path{
			case "/":
				fmt.Fprintf(w, "hello.")
			case "/execute": {

				var cmdParam string
				var inFileStr string
				var outTiffStr string
				var outTxtStr string
				var outGeoJStr string
				var usePz string

// might be time to start looking into that "help" thing.

				if r.Method == "GET" {
					cmdParam = r.URL.Query().Get("cmd")
					inFileStr = r.URL.Query().Get("inFiles")
					outTiffStr = r.URL.Query().Get("outTiffs")
					outTxtStr = r.URL.Query().Get("outTxts")
					outGeoJStr = r.URL.Query().Get("outGeoJson")
					usePz = r.URL.Query().Get("pz")
				} else {
					cmdParam = r.FormValue("cmd")
					inFileStr = r.FormValue("inFiles")
					outTiffStr = r.FormValue("outTiffs")
					outTxtStr = r.FormValue("outTxts")
					outGeoJStr = r.FormValue("outGeoJson")
					usePz = r.FormValue("pz")
				}

				cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
				cmdParamSlice := splitOrNil(cmdParam, " ")
				cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

				inFileSlice := splitOrNil(inFileStr, ",")
				outTiffSlice := splitOrNil(outTiffStr, ",")
				outTxtSlice := splitOrNil(outTxtStr, ",")
				outGeoJSlice := splitOrNil(outGeoJStr, ",")

				var output OutStruct
				
				if len(inFileSlice) > 0 { output.InFiles = make(map[string]string) }
				if len(outTiffSlice) + len(outTxtSlice) + len(outGeoJSlice)  > 0 {
					output.OutFiles = make(map[string]string)
				}

				for _, inFile := range inFileSlice {

					fName, err := pzsvc.Download(inFile, configObj.PzAddr)
					if err != nil {
						fmt.Fprintf(w, err.Error())
					} else {
						output.InFiles[inFile] = fName
					}
				}
				
				if len(cmdSlice) == 0 {
					fmt.Fprintf(w, `No cmd specified in config file.  Please provide "cmd" param.`)
					break
				}

				clc := exec.Command(cmdSlice[0], cmdSlice[1:]...)

				var b bytes.Buffer
				clc.Stdout = &b
				clc.Stderr = os.Stderr

				err = clc.Run()
				if err != nil { fmt.Fprintf(w, err.Error()) }

				output.ProgReturn = b.String()

				for _, outTiff := range outTiffSlice {
					dataId, err := pzsvc.IngestTiff(outTiff, configObj.PzAddr, cmdSlice[0])
					if err != nil {
						fmt.Fprintf(w, err.Error())
					} else {
						output.OutFiles[outTiff] = dataId
					}
				}

				for _, outTxt := range outTxtSlice {
					dataId, err := pzsvc.IngestTxt(outTxt, configObj.PzAddr, cmdSlice[0])
					if err != nil {
						fmt.Fprintf(w, err.Error())
					} else {
						output.OutFiles[outTxt] = dataId
					}
				}

				for _, outGeoJ := range outGeoJSlice {
					dataId, err := pzsvc.IngestGeoJson(outGeoJ, configObj.PzAddr, cmdSlice[0])
					if err != nil {
						fmt.Fprintf(w, err.Error())
					} else {
						output.OutFiles[outGeoJ] = dataId
					}
				}

				outBuf, err := json.Marshal(output)
				if err != nil { fmt.Fprintf(w, err.Error()) }
				
				outStr := string(outBuf)

				if usePz != "" {
					outStr = strconv.QuoteToASCII(outStr)
// TODO: clean this up a bit, and possibly move it back into
// the support function.
// - possibly include metadata to help on results searches?  Talk with Marge on where/how to put it in.
					outStr = fmt.Sprintf ( `{ "dataType": { "type": "text", "content": "%s" "mimeType": "text/plain" }, "metadata": {} }`, outStr )
				}

				fmt.Fprintf(w, outStr)
				
			}
			case "/description":
				if configObj.Description == "" {
					fmt.Fprintf (w, "No description defined")
				} else {
					fmt.Fprintf(w, configObj.Description)
				}
			case "/attributes":
				if configObj.Attributes == "" {
					fmt.Fprintf (w, "{ }")
				} else {
// convert attributes back into Json
// this might require specifying the interface a bit better.
//					fmt.Fprintf(w, configObj.Attributes)
				}
			case "/help":
				fmt.Fprintf(w, "We're sorry, help is not yet implemented.\n")
			default:
				fmt.Fprintf(w, "Command undefined.  Try help?\n")
		}
	})

	if configObj.Port <= 0 { configObj.Port = 8080 }

	portStr := ":" + strconv.Itoa(configObj.Port)

	log.Fatal(http.ListenAndServe(portStr, nil))
}

func splitOrNil(inString, knife string) []string {
fmt.Printf("SplitOrNull: \"%s\", split by \"%s\".\n", inString, knife)
	if inString == "" {
		return nil
	}
	return strings.Split(inString, knife)
}