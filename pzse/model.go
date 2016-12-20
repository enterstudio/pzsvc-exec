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

package pzse

import "github.com/venicegeo/pzsvc-exec/pzsvc"

// ConfigType represents and contains the information from a pzsvc-exec config file.
type ConfigType struct {
	CliCmd        string            // The first segment of the command to send to the CLI.  Security vulnerability when blank.
	VersionCmd    string            // The command to run to determine the version number of the underlying CLI.  Redundant with VersionStr
	VersionStr    string            // The version numebr of the underlyign CLI.  Redundant with VersionCmd
	PzAddr        string            // Address of local Piazza instance.  Used for Piazza file access.  Necessary for autoregistration, task worker
	AuthEnVar     string            // The environment variable containing the authorization token for the local Piazza instance.  Used similarly.
	SvcName       string            // The name to given for this service when registering.  Necessary for autoregistration, task worker.
	URL           string            // URL to give when registering.  Required when registering and not using task manager.
	Port          int               // Port to publish this service on.  Defaults to 8080.
	Description   string            // description to return when asked.
	Attributes    map[string]string // Service attributes.  Used to improve searching/sorting of services.
	NumProcs      int               // Number of jobs a single instance of this service can handle simultaneously
	CanUpload     bool              // True if this service is permitted to upload files
	CanDownlPz    bool              // True if this service is permitted to download files from Piazza
	CanDownlExt   bool              // True if this service is permitted to download files from an external source
	RegForTaskMgr bool              // True if sutoregistration shoudl be as a service using the Pz task manager
	MaxRunTime    int               // Time in seconds before a running job should be considered to have failed
	LocalOnly     bool              // True if service should only accept connections from localhost (used with task worker)
	JwtSecAuthURL string            // URL for taskworker to decrypt JWT.  If nonblank, will assume that all jobs are Beachfront JWT format
	LogAudit      bool              // True to log all auditable events
}

// OutStruct populates and provides the format for pzsvc-exec's output
type OutStruct struct {
	InFiles    map[string]string `json:"InFiles,omitempty"`
	OutFiles   map[string]string `json:"OutFiles,omitempty"`
	ProgStdOut string            `json:"ProgStdOut,omitempty"`
	ProgStdErr string            `json:"ProgStdErr,omitempty"`
	Errors     []string          `json:"Errors,omitempty"`
	HTTPStatus int               `json:"HTTPStatus,omitempty"`
}

// ConfigParseOut is a handy struct to organize all of the outputs
// for pzse.ConfigParse() and prevent potential confusion.
type ConfigParseOut struct {
	AuthKey  string
	PortStr  string
	Version  string
	ProcPool pzsvc.Semaphore
}

// InpStruct is the format that pzsvc-exec demarshals input data into
type InpStruct struct {
	Command    string   `json:"cmd,omitempty"`
	UserID     string   `json:"userID,omitempty"`       // string: unique ID of initiating user
	InPzFiles  []string `json:"inPzFiles,omitempty"`    // slice: Pz dataIds
	InExtFiles []string `json:"inExtFiles,omitempty"`   // slice: external URL
	InPzNames  []string `json:"inPzNames,omitempty"`    // slice: name for the InPzFile of the same index
	InExtNames []string `json:"inExtNames,omitempty"`   // slice: name for the InExtFile of the same index
	OutTiffs   []string `json:"outTiffs,omitempty"`     // slice: filenames of GeoTIFFs to be ingested
	OutTxts    []string `json:"outTxts,omitempty"`      // slice: filenames of text files to be ingested
	OutGeoJs   []string `json:"outGeoJson,omitempty"`   // slice: filenames of GeoJSON files to be ingested
	ExtAuth    string   `json:"inExtAuthKey,omitempty"` // string: auth key for accessing external files
	PzAuth     string   `json:"pzAuthKey,omitempty"`    // string: auth key for accessing Piazza
	PzAddr     string   `json:"pzAddr,omitempty"`       // string: URL for the targeted Pz instance
}

type rangeFunc func(string, string, string) (string, error)
