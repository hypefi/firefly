// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"net/http"

	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/oapispec"
	"github.com/hyperledger/firefly/pkg/fftypes"
)

var putSubscription = &oapispec.Route{
	Name:   "putSubscription",
	Path:   "namespaces/{ns}/subscriptions",
	Method: http.MethodPut,
	PathParams: []*oapispec.PathParam{
		{Name: "ns", ExampleFromConf: config.NamespacesDefault, Description: i18n.MsgTBD},
	},
	QueryParams:     nil,
	FilterFactory:   nil,
	Description:     i18n.MsgTBD,
	JSONInputValue:  func() interface{} { return &fftypes.Subscription{} },
	JSONOutputValue: func() interface{} { return &fftypes.Subscription{} },
	JSONOutputCodes: []int{http.StatusOK}, // Sync operation
	JSONInputSchema: newSubscriptionSchemaGenerator,
	JSONHandler: func(r *oapispec.APIRequest) (output interface{}, err error) {
		output, err = r.Or.CreateUpdateSubscription(r.Ctx, r.PP["ns"], r.Input.(*fftypes.Subscription))
		return output, err
	},
}
