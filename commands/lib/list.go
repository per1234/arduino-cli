// This file is part of arduino-cli.
//
// Copyright 2020 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package lib

import (
	"context"
	"strings"

	"github.com/arduino/arduino-cli/arduino"
	"github.com/arduino/arduino-cli/arduino/cores"
	"github.com/arduino/arduino-cli/arduino/libraries"
	"github.com/arduino/arduino-cli/arduino/libraries/librariesindex"
	"github.com/arduino/arduino-cli/arduino/libraries/librariesmanager"
	"github.com/arduino/arduino-cli/arduino/libraries/librariesresolver"
	"github.com/arduino/arduino-cli/commands"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
)

type installedLib struct {
	Library   *libraries.Library
	Available *librariesindex.Release
}

// LibraryList FIXMEDOC
func LibraryList(ctx context.Context, req *rpc.LibraryListRequest) (*rpc.LibraryListResponse, error) {
	pme, release := commands.GetPackageManagerExplorer(req)
	if pme == nil {
		return nil, &arduino.InvalidInstanceError{}
	}
	defer release()

	lm := commands.GetLibraryManager(req)
	if lm == nil {
		return nil, &arduino.InvalidInstanceError{}
	}

	nameFilter := strings.ToLower(req.GetName())

	var allLibs []*installedLib
	if fqbnString := req.GetFqbn(); fqbnString != "" {
		allLibs = listLibraries(lm, req.GetUpdatable(), true)
		fqbn, err := cores.ParseFQBN(req.GetFqbn())
		if err != nil {
			return nil, &arduino.InvalidFQBNError{Cause: err}
		}
		_, boardPlatform, _, _, refBoardPlatform, err := pme.ResolveFQBN(fqbn)
		if err != nil {
			return nil, &arduino.UnknownFQBNError{Cause: err}
		}

		filteredRes := map[string]*installedLib{}
		for _, lib := range allLibs {
			if cp := lib.Library.ContainerPlatform; cp != nil {
				if cp != boardPlatform && cp != refBoardPlatform {
					// Filter all libraries from extraneous platforms
					continue
				}
			}
			if latest, has := filteredRes[lib.Library.Name]; has {
				latestPriority := librariesresolver.ComputePriority(latest.Library, "", fqbn.PlatformArch)
				libPriority := librariesresolver.ComputePriority(lib.Library, "", fqbn.PlatformArch)
				if latestPriority >= libPriority {
					// Pick library with the best priority
					continue
				}
			}

			// Check if library is compatible with board specified by FBQN
			lib.Library.CompatibleWith = map[string]bool{
				fqbnString: lib.Library.IsCompatibleWith(fqbn.PlatformArch),
			}

			filteredRes[lib.Library.Name] = lib
		}

		allLibs = []*installedLib{}
		for _, lib := range filteredRes {
			allLibs = append(allLibs, lib)
		}
	} else {
		allLibs = listLibraries(lm, req.GetUpdatable(), req.GetAll())
	}

	installedLibs := []*rpc.InstalledLibrary{}
	for _, lib := range allLibs {
		if nameFilter != "" && strings.ToLower(lib.Library.Name) != nameFilter {
			continue
		}
		var release *rpc.LibraryRelease
		if lib.Available != nil {
			release = lib.Available.ToRPCLibraryRelease()
		}
		rpcLib, err := lib.Library.ToRPCLibrary()
		if err != nil {
			return nil, &arduino.PermissionDeniedError{Message: tr("Error getting information for library %s", lib.Library.Name), Cause: err}
		}
		installedLibs = append(installedLibs, &rpc.InstalledLibrary{
			Library: rpcLib,
			Release: release,
		})
	}

	return &rpc.LibraryListResponse{InstalledLibraries: installedLibs}, nil
}

// listLibraries returns the list of installed libraries. If updatable is true it
// returns only the libraries that may be updated.
func listLibraries(lm *librariesmanager.LibrariesManager, updatable bool, all bool) []*installedLib {
	res := []*installedLib{}
	for _, libAlternatives := range lm.Libraries {
		for _, lib := range libAlternatives {
			if !all {
				if lib.Location != libraries.User {
					continue
				}
			}
			available := lm.Index.FindLibraryUpdate(lib)
			if updatable && available == nil {
				continue
			}
			res = append(res, &installedLib{
				Library:   lib,
				Available: available,
			})
		}
	}
	return res
}
