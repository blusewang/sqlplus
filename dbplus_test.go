// Copyright 2019 MQ, Inc. All rights reserved.
//
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file in the root of the source
// tree.

package sqlplus

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestOpen(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	log.Println(rand.Intn(3))
}
