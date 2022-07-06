//    Copyright 2022 Ewout Prangsma
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package manager

import (
	api "github.com/binkynet/BinkyNet/apis/v1"
)

type ModuleFilter string

// Matches returns true if the given address is contained in the module with
// given module ID or the given module ID is empty or the address has a global module ID.
func (filter ModuleFilter) Matches(address api.ObjectAddress) bool {
	if filter == "" {
		return true
	}
	m, _, _ := api.SplitAddress(address)
	return m == string(filter) || m == api.GlobalModuleID
}

// MatchesModuleID
func (filter ModuleFilter) MatchesModuleID(id string) bool {
	if filter == "" {
		return true
	}
	return id == string(filter) || id == api.GlobalModuleID
}
