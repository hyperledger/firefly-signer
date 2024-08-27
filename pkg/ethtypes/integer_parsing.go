// Copyright © 2024 Kaleido, Inc.
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

package ethtypes

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"

	"github.com/hyperledger/firefly-common/pkg/i18n"
	"github.com/hyperledger/firefly-common/pkg/log"
	"github.com/hyperledger/firefly-signer/internal/signermsgs"
)

func BigIntegerFromString(ctx context.Context, s string) (*big.Int, error) {
	// We use Go's default '0' base integer parsing, where `0x` means hex,
	// no prefix means decimal etc.
	i, ok := new(big.Int).SetString(s, 0)
	if !ok {
		f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
		if err != nil {
			log.L(ctx).Errorf("Error parsing numeric string '%s': %s", s, err)
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidNumberString, s)
		}
		i, accuracy := f.Int(i)
		if accuracy != big.Exact {
			// If we weren't able to decode without losing precision, return an error
			return nil, i18n.NewError(ctx, signermsgs.MsgInvalidIntPrecisionLoss, s)
		}

		return i, nil
	}
	return i, nil
}

func UnmarshalBigInt(ctx context.Context, b []byte) (*big.Int, error) {
	var i interface{}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	err := d.Decode(&i)
	if err != nil {
		return nil, err
	}
	switch i := i.(type) {
	case json.Number:
		return BigIntegerFromString(context.Background(), i.String())
	case string:
		return BigIntegerFromString(context.Background(), i)
	default:
		return nil, i18n.NewError(ctx, signermsgs.MsgInvalidJSONTypeForBigInt, i)
	}
}
