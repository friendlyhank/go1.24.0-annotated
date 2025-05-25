// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// See map.go comment on the need for this routine.
TEXT ·mapinitnoop<ABIInternal>(SB),NOSPLIT,$0-0
	RET

