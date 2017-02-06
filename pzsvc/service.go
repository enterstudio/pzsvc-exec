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

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// FindMySvc Searches Pz for a service matching the input information.  If it finds
// one, it returns the service ID.  If it does not, returns an empty string.  Currently
// searches on service name and submitting user.
func FindMySvc(s Session, svcName string) (string, LoggedError) {
	query := s.PzAddr + "/service/me?per_page=1000&keyword=" + url.QueryEscape(svcName)
	var respObj SvcList
	LogAudit(s, s.UserID, "http request - looking for service "+svcName, query, "", INFO)
	byts, err := RequestKnownJSON("GET", "", query, s.PzAuth, &respObj)
	LogAuditBuf(s, query, "http response to service listing request", string(byts), s.UserID)
	if err != nil {
		return "", err.Log(s, "Error when finding Pz Service")
	}

	for _, checkServ := range respObj.Data {
		if checkServ.ResMeta.Name == svcName {
			return checkServ.ServiceID, nil
		}
	}

	return "", nil
}

// ManageRegistration Handles Pz registration for a service.  It checks the current
// service list to see if it has been registered already.  If it has not, it performs
// initial registration.  If it has not, it re-registers.  Best practice is to do this
// every time your service starts up.  For those of you code-reading, the filter is
// still somewhat rudimentary.  It will improve as better tools become available.
func ManageRegistration(s Session, svcObj Service) LoggedError {

	var pzErr *Error
	var resp *http.Response
	LogInfo(s, "Searching for service in Pz service list")
	svcID, err := FindMySvc(s, svcObj.ResMeta.Name)
	if err != nil {
		return err
	}

	svcJSON, err := json.Marshal(svcObj)

	if svcID == "" {
		LogInfo(s, "Registering Service")
		targURL := s.PzAddr + "/service"
		LogAuditBuf(s, s.AppName, "Registering Service request", string(svcJSON), targURL)
		resp, pzErr = SubmitSinglePart("POST", string(svcJSON), targURL, s.PzAuth)
		LogAuditResponse(s, targURL, "Registering Service Response", resp, s.AppName)
	} else {
		LogInfo(s, "Updating Service Registration")
		targURL := s.PzAddr + "/service/" + svcID
		LogAuditBuf(s, s.AppName, "Updating Service request", string(svcJSON), targURL)
		resp, pzErr = SubmitSinglePart("PUT", string(svcJSON), s.PzAddr+"/service/"+svcID, s.PzAuth)
		LogAuditResponse(s, targURL, "Updating Service Response", resp, s.AppName)
	}
	if pzErr != nil {
		return pzErr.Log(s, "Error managing registration: ")
	}

	return nil
}
