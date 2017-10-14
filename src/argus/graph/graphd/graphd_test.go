// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-10 20:45 (EDT)
// Function:

package graphd

import (
	"encoding/binary"
	"testing"

	"argus/diag"
)

func TestGraphd(t *testing.T) {

	diag.SetConfig(&diag.Config{Debug: map[string]bool{"graphd": true}})

	g := open("/tmp/tngdata/gdata/W/N/Top:Test:vr0:out")
	if g == nil {
		diag.Fatal("open failed")
	}
	defer g.close()
	diag.Verbose("g %+v", g)
	diag.Verbose("h %+v", g.h)

	pt := uint32(0)

	for i := 0; i < int(g.h.Samp.NMax); i++ {
		g.f.Seek(g.sampStart+int64(i)*SampSize, 0)

		s := &SampleData{}
		binary.Read(g.f, binary.BigEndian, s)

		//diag.Verbose("[%4d] %d %+v", i, s.When-pt, s)

		pt = s.When

	}

	r := NewCbufReader(g.f, g.sampStart, int64(g.h.Samp.NMax*SampSize))
	r.Seek(int64(g.h.Samp.Idx) * int64(SampSize))

	for i := 0; i < int(g.h.Samp.NMax); i++ {
		s := &SampleData{}
		binary.Read(r, binary.BigEndian, s)

		diag.Verbose("[%4d] %d %+v", i, s.When-pt, s)

		pt = s.When
	}

}

/*

[   0] 376905881 &{When:376905881 Status:1 Value:912.64484 Exp:0 Delt:0}
[   1] 30 &{When:376905911 Status:1 Value:352.38815 Exp:0 Delt:0}

[ 200] 30 &{When:376911881 Status:1 Value:13263.591 Exp:0 Delt:0}
[ 201] 4294936531 &{When:376881116 Status:1 Value:4158.568 Exp:0 Delt:0}
[ 202] 30 &{When:376881146 Status:1 Value:3129.6887 Exp:0 Delt:0}
[ 203] 30 &{When:376881176 Status:1 Value:3290.7117 Exp:0 Delt:0}

[1022] 30 &{When:376905821 Status:1 Value:1927.1721 Exp:0 Delt:0}
[1023] 30 &{When:376905851 Status:1 Value:760.2489 Exp:0 Delt:0}


===

[   0] 4294942651 &{When:376881206 Status:1 Value:9249.409 Exp:0 Delt:0}
[   1] 30 &{When:376881236 Status:1 Value:9345.861 Exp:0 Delt:0}
[   2] 30 &{When:376881266 Status:1 Value:7382.1143 Exp:0 Delt:0}

[ 818] 30 &{When:376905821 Status:1 Value:1927.1721 Exp:0 Delt:0}
[ 819] 30 &{When:376905851 Status:1 Value:760.2489 Exp:0 Delt:0}
[ 820] 4294937326 &{When:376875881 Status:1 Value:0 Exp:6848.114 Delt:2735.1348}
[ 821] 780713627 &{When:1157589508 Status:0 Value:0 Exp:1.9926919e-25 Delt:1e-45}

*/
