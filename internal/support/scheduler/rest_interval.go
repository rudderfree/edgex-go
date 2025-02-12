/*******************************************************************************
 * Copyright 2019 VMware Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package scheduler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/edgexfoundry/edgex-go/internal/pkg"
	"github.com/edgexfoundry/edgex-go/internal/support/scheduler/errors"
	"github.com/edgexfoundry/edgex-go/internal/support/scheduler/operators/interval"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/types"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/gorilla/mux"
)

func restGetIntervals(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	op := interval.NewAllExecutor(dbClient, Configuration.Service.MaxResultCount)
	intervals, err := op.Execute()
	if err != nil {
		LoggingClient.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pkg.Encode(intervals, w, LoggingClient)
}

func restUpdateInterval(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	var from models.Interval
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&from)

	// Problem decoding
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		LoggingClient.Error("Error decoding the interval: " + err.Error())
		return
	}

	LoggingClient.Info("Updating Interval: " + from.ID)
	op := interval.NewUpdateExecutor(dbClient, scClient, from)
	err = op.Execute()
	if err != nil {
		switch t := err.(type) {
		case errors.ErrIntervalNotFound:
			http.Error(w, t.Error(), http.StatusNotFound)
		case errors.ErrInvalidCronFormat:
			http.Error(w, t.Error(), http.StatusBadRequest)
		case errors.ErrIntervalStillUsedByIntervalActions:
			http.Error(w, t.Error(), http.StatusBadRequest)
		case *errors.ErrIntervalNameInUse:
			http.Error(w, t.Error(), http.StatusBadRequest)
		default: //return an error on everything else.
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		LoggingClient.Error(err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("true"))
}

func restAddInterval(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	var intervalObj models.Interval
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&intervalObj)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		LoggingClient.Error("Error decoding interval" + err.Error())
		return
	}
	LoggingClient.Info("Posting new Interval: " + intervalObj.String())

	op := interval.NewAddExecutor(dbClient, scClient, intervalObj)
	newId, err := op.Execute()
	if err != nil {
		switch t := err.(type) {
		case *errors.ErrIntervalNameInUse:
			http.Error(w, t.Error(), http.StatusBadRequest)
		default:
			http.Error(w, t.Error(), http.StatusInternalServerError)
		}
		LoggingClient.Error(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(newId))
}

func restGetIntervalByID(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	// URL parameters
	vars := mux.Vars(r)
	id, err := url.QueryUnescape(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		LoggingClient.Error("Error un-escaping the value id: " + err.Error())
		return
	}

	op := interval.NewIdExecutor(dbClient, id)
	result, err := op.Execute()
	if err != nil {
		LoggingClient.Error(err.Error())
		switch err.(type) {
		case errors.ErrIntervalNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	pkg.Encode(result, w, LoggingClient)
}

func restDeleteIntervalByID(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	// URL parameters
	vars := mux.Vars(r)
	id, err := url.QueryUnescape(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		LoggingClient.Error("Error un-escaping the value id: " + err.Error())
		return
	}

	op := interval.NewDeleteByIDExecutor(dbClient, scClient, id)
	err = op.Execute()

	if err != nil {
		LoggingClient.Error(err.Error())
		switch err.(type) {
		case errors.ErrIntervalNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("true"))
}
func restGetIntervalByName(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	name, err := url.QueryUnescape(vars["name"])

	//Issues un-escaping
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		LoggingClient.Error("Error un-escaping the value name: " + err.Error())
		return
	}

	op := interval.NewNameExecutor(dbClient, name)
	result, err := op.Execute()
	if err != nil {
		switch err := err.(type) {
		case errors.ErrIntervalNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		case *types.ErrServiceClient:
			http.Error(w, err.Error(), err.StatusCode)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		LoggingClient.Error(err.Error())
		return
	}

	pkg.Encode(result, w, LoggingClient)

}

func restDeleteIntervalByName(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	vars := mux.Vars(r)
	name, err := url.QueryUnescape(vars["name"])

	//Issues un-escaping
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		LoggingClient.Error("Error un-escaping the value name: " + err.Error())
		return
	}
	op := interval.NewDeleteByNameExecutor(dbClient, scClient, name)
	err = op.Execute()
	if err != nil {

		switch err.(type) {
		case errors.ErrIntervalNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		case errors.ErrIntervalStillUsedByIntervalActions:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("true"))
}

// ************************ UTILITY HANDLERS ************************************

// Scrub all the Intervals and IntervalActions
func restScrubAllIntervals(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	LoggingClient.Info("Scrubbing All Interval(s) and IntervalAction(s).")
	op := interval.NewScrubExecutor(dbClient)
	count, err := op.Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(count)))

}
