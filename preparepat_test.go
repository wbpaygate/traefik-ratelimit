package traefik_ratelimit

import (
//	"fmt"
	"testing"
)


func Test_preparepat(t *testing.T) {
	cases := []struct {
		name string
		ipt  []int
		s    string
		resb bool
		ress string
	} {
		{
			name: "t1",
			ipt:  []int{1,2,3},
			s:    "/",
			resb: false,
			ress: "",
		},
		{
			name: "t1",
			ipt:  []int{1,-2,-1},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "1:aa/-2:cc/-1:dd",
		},
		{
			name: "t1",
			ipt:  []int{-4,-2,-1},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "-4:aa/-2:cc/-1:dd",
		},
		{
			name: "t1",
			ipt:  []int{-5,-2,-1},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "-5:/-2:cc/-1:dd",
		},
		{
			name: "t1",
			ipt:  []int{-6,-2,-1},
			s:    "/aa/bb/cc/dd",
			resb: false,
			ress: "",
		},

		{
			name: "t1",
			ipt:  []int{6},
			s:    "/aa/bb/cc/dd",
			resb: false,
			ress: "",
		},

		{
			name: "t1",
			ipt:  []int{4},
			s:    "/aa/bb/cc/dd",
			resb: true,
			ress: "4:dd",
		},

	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ress, resb := preparepat(tc.ipt, tc.s)
			if ress != tc.ress || resb != tc.resb {
				t.Errorf("preparepat %s expect %q %q , %v %v", tc.name, ress, tc.ress, resb, tc.resb)
				return
			}
		})
	}
}


