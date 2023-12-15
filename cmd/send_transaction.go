// Copyright © 2023 Kaleido, Inc.
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

package cmd

import (
	"github.com/spf13/cobra"
)

var transactionFile string

func sendTransactionCommand() *cobra.Command {
	sendTransactionCmd := &cobra.Command{
		Use:   "send-transaction input-file.json",
		Short: "Submits a transaction from a JSON file",
		Long: `The JSON input includes the following parameters:
- abi: One or more function definitions
- method: The name of the function to invoke - required if the ABI contains more than one function
- params: The input parameters, as an object, or array
As well as ethereum signing parameters
- from: The ethereum address to use to sign - must already configured in the wallet
- to: The contract address to invoke (required - as this cannot be used for contract deploy)
- nonce: The nonce for the transaction (required)
- gas: The maximum gas limit for execution of the transaction (required)
- gasPrice
- maxPriorityFeePerGas
- maxFeePerGas
- value
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := initServer()
			if err != nil {
				return err
			}
			return server.SignTransactionFromFile(cmd.Context(), transactionFile)
		},
	}
	sendTransactionCmd.Flags().StringVarP(&transactionFile, "input", "i", "", "input file")
	_ = sendTransactionCmd.MarkFlagRequired("input")
	return sendTransactionCmd
}
