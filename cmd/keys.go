/*
Copyright © 2020 Jack Zampolin jack.zampolin@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	ckeys "github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/relayer/helpers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

// keysCmd represents the keys command
func keysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Aliases: []string{"k"},
		Short:   "manage keys held by the relayer for each chain",
	}

	cmd.AddCommand(keysAddCmd())
	cmd.AddCommand(keysRestoreCmd())
	cmd.AddCommand(keysDeleteCmd())
	cmd.AddCommand(keysListCmd())
	cmd.AddCommand(keysShowCmd())
	cmd.AddCommand(keysExportCmd())

	return cmd
}

// keysAddCmd respresents the `keys add` command
func keysAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [chain-id] [[name]]",
		Aliases: []string{"a"},
		Short:   "adds a key to the keychain associated with a particular chain",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, err := config.Chains.Get(args[0])
			if err != nil {
				return err
			}

			var keyName string
			if len(args) == 2 {
				keyName = args[1]
			} else {
				keyName = chain.Key
			}

			if chain.KeyExists(keyName) {
				return errKeyExists(keyName)
			}

			// Adding key with key add helper
			ko, err := helpers.KeyAddOrRestore(chain, keyName)
			if err != nil {
				return err
			}

			out, err := json.Marshal(&ko)
			if err != nil {
				return err
			}

			fmt.Println(string(out))
			return nil
		},
	}

	return cmd
}

// keysRestoreCmd respresents the `keys add` command
func keysRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restore [chain-id] [name] [mnemonic]",
		Aliases: []string{"r"},
		Short:   "restores a mnemonic to the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyName := args[1]
			chain, err := config.Chains.Get(args[0])
			if err != nil {
				return err
			}

			if chain.KeyExists(keyName) {
				return errKeyExists(keyName)
			}

			// Restoring key with passing mnemonic
			ko, err := helpers.KeyAddOrRestore(chain, keyName, args[2])
			if err != nil {
				return err
			}

			fmt.Println(ko.Address)
			return nil
		},
	}

	return cmd
}

// keysDeleteCmd respresents the `keys delete` command
func keysDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [chain-id] [[name]]",
		Aliases: []string{"d"},
		Short:   "deletes a key from the keychain associated with a particular chain",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, err := config.Chains.Get(args[0])
			if err != nil {
				return err
			}

			var keyName string
			if len(args) == 2 {
				keyName = args[1]
			} else {
				keyName = chain.Key
			}

			if !chain.KeyExists(keyName) {
				return errKeyDoesntExist(keyName)
			}

			// TODO: prompt to delete with flag to ignore

			err = chain.Keybase.Delete(keyName)
			if err != nil {
				panic(err)
			}

			fmt.Printf("key %s deleted\n", keyName)
			return nil
		},
	}

	return cmd
}

// keysListCmd respresents the `keys list` command
func keysListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [chain-id]",
		Aliases: []string{"l"},
		Short:   "lists keys from the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, err := config.Chains.Get(args[0])
			if err != nil {
				return err
			}

			info, err := chain.Keybase.List()
			if err != nil {
				return err
			}

			for d, i := range info {
				fmt.Printf("key(%d): %s -> %s\n", d, i.GetName(), i.GetAddress().String())
			}

			return nil
		},
	}

	return cmd
}

// keysShowCmd respresents the `keys show` command
func keysShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [chain-id] [[name]]",
		Aliases: []string{"s"},
		Short:   "shows a key from the keychain associated with a particular chain",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			chain, err := config.Chains.Get(args[0])
			if err != nil {
				return err
			}

			var keyName string
			if len(args) == 2 {
				keyName = args[1]
			} else {
				keyName = chain.Key
			}

			if !chain.KeyExists(keyName) {
				return errKeyDoesntExist(keyName)
			}

			info, err := chain.Keybase.Key(keyName)
			if err != nil {
				return err
			}

			fmt.Println(info.GetAddress().String())
			return nil
		},
	}

	return cmd
}

// keysExportCmd respresents the `keys export` command
func keysExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "export [chain-id] [name]",
		Aliases: []string{"e"},
		Short:   "exports a privkey from the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyName := args[1]
			chain, err := config.Chains.Get(args[0])
			if err != nil {
				return err
			}

			if !chain.KeyExists(keyName) {
				return errKeyDoesntExist(keyName)
			}

			info, err := chain.Keybase.ExportPrivKeyArmor(keyName, ckeys.DefaultKeyPass)
			if err != nil {
				return err
			}

			fmt.Println(info)
			return nil
		},
	}

	return cmd
}

// API Handlers

type keyResponse struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func formatKey(info keyring.Info) keyResponse {
	return keyResponse{
		Name:    info.GetName(),
		Address: info.GetAddress().String(),
	}
}

// GetKeysHandler handles the route
func GetKeysHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chain, err := config.Chains.Get(vars["chain-id"])
	if err != nil {
		helpers.WriteErrorResponse(http.StatusBadRequest, err, w)
		return
	}
	info, err := chain.Keybase.List()
	if err != nil {
		helpers.WriteErrorResponse(http.StatusInternalServerError, err, w)
		return
	}

	keys := make([]keyResponse, len(info))
	for index, key := range info {
		keys[index] = formatKey(key)
	}
	helpers.SuccessJSONResponse(http.StatusOK, keys, w)
}

// GetKeyHandler handles the route
func GetKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chain, err := config.Chains.Get(vars["chain-id"])
	if err != nil {
		helpers.WriteErrorResponse(http.StatusBadRequest, err, w)
		return
	}

	keyName := vars["name"]
	if !chain.KeyExists(keyName) {
		helpers.WriteErrorResponse(http.StatusNotFound, errKeyDoesntExist(keyName), w)
		return
	}

	info, err := chain.Keybase.Key(keyName)
	if err != nil {
		helpers.WriteErrorResponse(http.StatusInternalServerError, err, w)
		return
	}
	helpers.SuccessJSONResponse(http.StatusOK, formatKey(info), w)
}

// PostKeyHandler handles the route
func PostKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chain, err := config.Chains.Get(vars["chain-id"])
	if err != nil {
		helpers.WriteErrorResponse(http.StatusBadRequest, err, w)
		return
	}

	keyName := vars["name"]
	if chain.KeyExists(keyName) {
		helpers.WriteErrorResponse(http.StatusBadRequest, errKeyExists(keyName), w)
		return
	}

	ko, err := helpers.KeyAddOrRestore(chain, keyName)
	if err != nil {
		helpers.WriteErrorResponse(http.StatusInternalServerError, err, w)
		return
	}
	helpers.SuccessJSONResponse(http.StatusCreated, ko, w)
}

type restoreKeyRequest struct {
	Mnemonic string `json:"mnemonic"`
}

// RestoreKeyHandler handles the route
func RestoreKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chain, err := config.Chains.Get(vars["chain-id"])
	if err != nil {
		helpers.WriteErrorResponse(http.StatusBadRequest, err, w)
		return
	}

	keyName := vars["name"]
	if chain.KeyExists(keyName) {
		helpers.WriteErrorResponse(http.StatusNotFound, errKeyExists(keyName), w)
		return
	}

	var request restoreKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		helpers.WriteErrorResponse(http.StatusBadRequest, err, w)
		return
	}

	ko, err := helpers.KeyAddOrRestore(chain, keyName, request.Mnemonic)
	if err != nil {
		helpers.WriteErrorResponse(http.StatusInternalServerError, err, w)
		return
	}
	helpers.SuccessJSONResponse(http.StatusOK, ko, w)
}

// DeleteKeyHandler handles the route
func DeleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chain, err := config.Chains.Get(vars["chain-id"])
	if err != nil {
		helpers.WriteErrorResponse(http.StatusBadRequest, err, w)
		return
	}

	keyName := vars["name"]
	if !chain.KeyExists(keyName) {
		helpers.WriteErrorResponse(http.StatusNotFound, errKeyDoesntExist(keyName), w)
		return
	}

	err = chain.Keybase.Delete(keyName)
	if err != nil {
		helpers.WriteErrorResponse(http.StatusInternalServerError, err, w)
		return
	}
	helpers.SuccessJSONResponse(http.StatusOK, fmt.Sprintf("key %s deleted", keyName), w)
}
