// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 11:09 (EDT)
// Function:

package argus

import (
	"encoding/hex"
)

// Once upon a weekend weary, while I pondered, beat and bleary,
// Over many a faintly printed hexadecimal dump of core --
// While I nodded, nearly napping, suddenly there came a tapping,
// As of some Source user chatting, chatting of some Mavenlore.
// "Just a power glitch," I muttered, "printing out an underscore --
//                 Just a glitch and nothing more."
//   -- the Dragon, The Maven

func HexDump(x []byte) string {
	return hex.Dump(x)
}
