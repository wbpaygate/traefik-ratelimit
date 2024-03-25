package pat

import (
	//	"fmt"
	"testing"
)

func Test_preparepat(t *testing.T) {
	cases := []struct {
		name string
		ipt  []Pat
		s    string
		resb bool
		ress string
	}{

		{
			name: "t1",
			ipt:  []Pat{Pat{v: 1}, Pat{v: 2}, Pat{v: 3}},
			s:    "/",
			resb: false,
			ress: "",
		},

		{
			name: "t2",
			ipt:  []Pat{Pat{v: 1}, Pat{v: -2}, Pat{v: -1}},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "1:aa/-2:cc/-1:dd",
		},

		{
			name: "t3",
			ipt:  []Pat{Pat{t: 2, v: 0}},
			s:    "/aa/bb/cc/dd",
			resb: false,
			ress: "",
		},

		{
			name: "t4",
			ipt:  []Pat{Pat{v: -4}, Pat{v: -2}, Pat{v: -1}},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "-4:aa/-2:cc/-1:dd",
		},

		{
			name: "t5",
			ipt:  []Pat{Pat{v: -5}, Pat{v: -2}, Pat{v: -1}},
			s:    "/aa/bb/cc/dd",
			resb: false,
			ress: "",
		},

		{
			name: "t6",
			ipt:  []Pat{Pat{t: 1, v: -3}, Pat{v: -2}, Pat{v: -1}},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "-3:*/-2:cc/-1:dd",
		},

		{
			name: "t7",
			ipt:  []Pat{Pat{t: 2, v: 6}},
			s:    "/aa/bb/cc/dd",
			resb: false,
			ress: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ress, resb := Preparepat(tc.ipt, tc.s)
			if ress != tc.ress || resb != tc.resb {
				t.Errorf("preparepat %s expect %q %q , %v %v", tc.name, ress, tc.ress, resb, tc.resb)
				return
			}
		})
	}
}
